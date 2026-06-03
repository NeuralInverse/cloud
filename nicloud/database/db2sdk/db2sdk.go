// Package db2sdk provides common conversion routines from database types to nicloudsdk types
package db2sdk

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/sqlc-dev/pqtype"
	"golang.org/x/xerrors"
	"tailscale.com/tailcfg"

	agentproto "github.com/NeuralInverse/cloud/v2/agent/proto"
	aibridgeutils "github.com/NeuralInverse/cloud/v2/aibridge/utils"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtime"
	"github.com/NeuralInverse/cloud/v2/nicloud/externalauth/gitprovider"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloud/render"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/slice"
	"github.com/NeuralInverse/cloud/v2/nicloud/workspaceapps/appurl"
	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd/chatprompt"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/tailnet"
	previewtypes "github.com/coder/preview/types"
)

func APIAllowListTarget(entry rbac.AllowListElement) nicloudsdk.APIAllowListTarget {
	return nicloudsdk.APIAllowListTarget{
		Type: nicloudsdk.RBACResource(entry.Type),
		ID:   entry.ID,
	}
}

// AIProvider converts a database row plus its API keys into the
// nicloudsdk shape. The caller is responsible for ensuring the row and
// keys have been decrypted (i.e. fetched through the dbcrypt-wrapped
// store). Each api_key is masked via aibridge utils.MaskSecret and
// write-only fields on Settings are stripped, so the result is safe
// to echo back in API responses.
func AIProvider(row database.AIProvider, keys []database.AIProviderKey) (nicloudsdk.AIProvider, error) {
	display := row.Name
	if row.DisplayName.Valid && row.DisplayName.String != "" {
		display = row.DisplayName.String
	}
	out := nicloudsdk.AIProvider{
		ID:          row.ID,
		Type:        nicloudsdk.AIProviderType(row.Type),
		Name:        row.Name,
		DisplayName: display,
		Enabled:     row.Enabled,
		BaseURL:     row.BaseUrl,
		APIKeys:     maskAIProviderKeys(keys),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	s, err := AIProviderSettings(row.Settings)
	if err != nil {
		return nicloudsdk.AIProvider{}, xerrors.Errorf("decode settings: %w", err)
	}
	out.Settings = redactAIProviderSettings(s)
	return out, nil
}

// AIProviderSettings parses the on-disk JSON form back into a nicloudsdk
// settings value. SQL NULL and the empty string decode to the zero
// value.
func AIProviderSettings(col sql.NullString) (nicloudsdk.AIProviderSettings, error) {
	if !col.Valid || col.String == "" {
		return nicloudsdk.AIProviderSettings{}, nil
	}
	var s nicloudsdk.AIProviderSettings
	if err := json.Unmarshal([]byte(col.String), &s); err != nil {
		return nicloudsdk.AIProviderSettings{}, err
	}
	return s, nil
}

// maskAIProviderKeys converts the supplied database rows into the
// public-facing AIProviderKey shape, preserving order. Plaintext is
// replaced by a non-reversible mask (see aibridgeutils.MaskSecret) so
// the result is safe to embed in API responses.
func maskAIProviderKeys(keys []database.AIProviderKey) []nicloudsdk.AIProviderKey {
	out := make([]nicloudsdk.AIProviderKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, nicloudsdk.AIProviderKey{
			ID:        k.ID,
			Masked:    aibridgeutils.MaskSecret(k.APIKey),
			CreatedAt: k.CreatedAt,
		})
	}
	return out
}

// redactAIProviderSettings strips write-only fields from a settings
// value so it can be safely echoed back in API responses.
func redactAIProviderSettings(s nicloudsdk.AIProviderSettings) nicloudsdk.AIProviderSettings {
	out := s
	if out.Bedrock != nil {
		// Deep-copy so we don't mutate the caller's struct.
		b := *out.Bedrock
		b.AccessKey = nil
		b.AccessKeySecret = nil
		out.Bedrock = &b
	}
	return out
}

type ExternalAuthMeta struct {
	Authenticated bool
	ValidateError string
}

func ExternalAuths(auths []database.ExternalAuthLink, meta map[string]ExternalAuthMeta) []nicloudsdk.ExternalAuthLink {
	out := make([]nicloudsdk.ExternalAuthLink, 0, len(auths))
	for _, auth := range auths {
		out = append(out, ExternalAuth(auth, meta[auth.ProviderID]))
	}
	return out
}

func ExternalAuth(auth database.ExternalAuthLink, meta ExternalAuthMeta) nicloudsdk.ExternalAuthLink {
	return nicloudsdk.ExternalAuthLink{
		ProviderID:      auth.ProviderID,
		CreatedAt:       auth.CreatedAt,
		UpdatedAt:       auth.UpdatedAt,
		HasRefreshToken: auth.OAuthRefreshToken != "",
		Expires:         auth.OAuthExpiry,
		Authenticated:   meta.Authenticated,
		ValidateError:   meta.ValidateError,
	}
}

func WorkspaceBuildParameter(p database.WorkspaceBuildParameter) nicloudsdk.WorkspaceBuildParameter {
	return nicloudsdk.WorkspaceBuildParameter{
		Name:  p.Name,
		Value: p.Value,
	}
}

func WorkspaceBuildParameters(params []database.WorkspaceBuildParameter) []nicloudsdk.WorkspaceBuildParameter {
	return slice.List(params, WorkspaceBuildParameter)
}

func TemplateVersionParameters(params []database.TemplateVersionParameter) ([]nicloudsdk.TemplateVersionParameter, error) {
	out := make([]nicloudsdk.TemplateVersionParameter, 0, len(params))
	for _, p := range params {
		np, err := TemplateVersionParameter(p)
		if err != nil {
			return nil, xerrors.Errorf("convert template version parameter %q: %w", p.Name, err)
		}
		out = append(out, np)
	}

	return out, nil
}

func TemplateVersionParameterFromPreview(param previewtypes.Parameter) (nicloudsdk.TemplateVersionParameter, error) {
	descriptionPlaintext, err := render.PlaintextFromMarkdown(param.Description)
	if err != nil {
		return nicloudsdk.TemplateVersionParameter{}, err
	}

	sdkParam := nicloudsdk.TemplateVersionParameter{
		Name:                 param.Name,
		DisplayName:          param.DisplayName,
		Description:          param.Description,
		DescriptionPlaintext: descriptionPlaintext,
		Type:                 string(param.Type),
		FormType:             string(param.FormType),
		Mutable:              param.Mutable,
		DefaultValue:         param.DefaultValue.AsString(),
		Icon:                 param.Icon,
		Required:             param.Required,
		Ephemeral:            param.Ephemeral,
		Options:              slice.List(param.Options, TemplateVersionParameterOptionFromPreview),
		// Validation set after
	}
	if len(param.Validations) > 0 {
		validation := param.Validations[0]
		sdkParam.ValidationError = validation.Error
		if validation.Monotonic != nil {
			sdkParam.ValidationMonotonic = nicloudsdk.ValidationMonotonicOrder(*validation.Monotonic)
		}
		if validation.Regex != nil {
			sdkParam.ValidationRegex = *validation.Regex
		}
		if validation.Min != nil {
			//nolint:gosec // No other choice
			sdkParam.ValidationMin = ptr.Ref(int32(*validation.Min))
		}
		if validation.Max != nil {
			//nolint:gosec // No other choice
			sdkParam.ValidationMax = ptr.Ref(int32(*validation.Max))
		}
	}

	return sdkParam, nil
}

func TemplateVersionParameter(param database.TemplateVersionParameter) (nicloudsdk.TemplateVersionParameter, error) {
	options, err := templateVersionParameterOptions(param.Options)
	if err != nil {
		return nicloudsdk.TemplateVersionParameter{}, err
	}

	descriptionPlaintext, err := render.PlaintextFromMarkdown(param.Description)
	if err != nil {
		return nicloudsdk.TemplateVersionParameter{}, err
	}

	var validationMin *int32
	if param.ValidationMin.Valid {
		validationMin = &param.ValidationMin.Int32
	}

	var validationMax *int32
	if param.ValidationMax.Valid {
		validationMax = &param.ValidationMax.Int32
	}

	return nicloudsdk.TemplateVersionParameter{
		Name:                 param.Name,
		DisplayName:          param.DisplayName,
		Description:          param.Description,
		DescriptionPlaintext: descriptionPlaintext,
		Type:                 param.Type,
		FormType:             string(param.FormType),
		Mutable:              param.Mutable,
		DefaultValue:         param.DefaultValue,
		Icon:                 param.Icon,
		Options:              options,
		ValidationRegex:      param.ValidationRegex,
		ValidationMin:        validationMin,
		ValidationMax:        validationMax,
		ValidationError:      param.ValidationError,
		ValidationMonotonic:  nicloudsdk.ValidationMonotonicOrder(param.ValidationMonotonic),
		Required:             param.Required,
		Ephemeral:            param.Ephemeral,
	}, nil
}

func MinimalUser(user database.User) nicloudsdk.MinimalUser {
	return nicloudsdk.MinimalUser{
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
	}
}

func MinimalUserFromVisibleUser(user database.VisibleUser) nicloudsdk.MinimalUser {
	return nicloudsdk.MinimalUser{
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
	}
}

func ReducedUser(user database.User) nicloudsdk.ReducedUser {
	return nicloudsdk.ReducedUser{
		MinimalUser:      MinimalUser(user),
		Email:            user.Email,
		CreatedAt:        user.CreatedAt,
		UpdatedAt:        user.UpdatedAt,
		LastSeenAt:       user.LastSeenAt,
		Status:           nicloudsdk.UserStatus(user.Status),
		LoginType:        nicloudsdk.LoginType(user.LoginType),
		IsServiceAccount: user.IsServiceAccount,
	}
}

func UserFromGroupMember(member database.GroupMember) database.User {
	return database.User{
		ID:                 member.UserID,
		Email:              member.UserEmail,
		Username:           member.UserUsername,
		HashedPassword:     member.UserHashedPassword,
		CreatedAt:          member.UserCreatedAt,
		UpdatedAt:          member.UserUpdatedAt,
		Status:             member.UserStatus,
		RBACRoles:          member.UserRbacRoles,
		LoginType:          member.UserLoginType,
		AvatarURL:          member.UserAvatarUrl,
		Deleted:            member.UserDeleted,
		LastSeenAt:         member.UserLastSeenAt,
		QuietHoursSchedule: member.UserQuietHoursSchedule,
		Name:               member.UserName,
		GithubComUserID:    member.UserGithubComUserID,
		IsServiceAccount:   member.UserIsServiceAccount,
	}
}

func ReducedUserFromGroupMember(member database.GroupMember) nicloudsdk.ReducedUser {
	return ReducedUser(UserFromGroupMember(member))
}

func ReducedUsersFromGroupMembers(members []database.GroupMember) []nicloudsdk.ReducedUser {
	return slice.List(members, ReducedUserFromGroupMember)
}

func UserFromGroupMemberRow(member database.GetGroupMembersByGroupIDPaginatedRow) database.User {
	return database.User{
		ID:                 member.UserID,
		Email:              member.UserEmail,
		Username:           member.UserUsername,
		HashedPassword:     member.UserHashedPassword,
		CreatedAt:          member.UserCreatedAt,
		UpdatedAt:          member.UserUpdatedAt,
		Status:             member.UserStatus,
		RBACRoles:          member.UserRbacRoles,
		LoginType:          member.UserLoginType,
		AvatarURL:          member.UserAvatarUrl,
		Deleted:            member.UserDeleted,
		LastSeenAt:         member.UserLastSeenAt,
		QuietHoursSchedule: member.UserQuietHoursSchedule,
		Name:               member.UserName,
		GithubComUserID:    member.UserGithubComUserID,
		IsServiceAccount:   member.UserIsServiceAccount,
	}
}

func ReducedUserFromGroupMemberRow(member database.GetGroupMembersByGroupIDPaginatedRow) nicloudsdk.ReducedUser {
	return ReducedUser(UserFromGroupMemberRow(member))
}

func ReducedUsersFromGroupMemberRows(members []database.GetGroupMembersByGroupIDPaginatedRow) []nicloudsdk.ReducedUser {
	return slice.List(members, ReducedUserFromGroupMemberRow)
}

func ReducedUsers(users []database.User) []nicloudsdk.ReducedUser {
	return slice.List(users, ReducedUser)
}

func User(user database.User, organizationIDs []uuid.UUID) nicloudsdk.User {
	convertedUser := nicloudsdk.User{
		ReducedUser:     ReducedUser(user),
		OrganizationIDs: organizationIDs,
		Roles:           SlimRolesFromNames(user.RBACRoles),
	}

	return convertedUser
}

func Users(users []database.User, organizationIDs map[uuid.UUID][]uuid.UUID) []nicloudsdk.User {
	return slice.List(users, func(user database.User) nicloudsdk.User {
		return User(user, organizationIDs[user.ID])
	})
}

func Group(row database.GetGroupsRow, members []database.GroupMember, totalMemberCount int) nicloudsdk.Group {
	return nicloudsdk.Group{
		ID:                      row.Group.ID,
		Name:                    row.Group.Name,
		DisplayName:             row.Group.DisplayName,
		OrganizationID:          row.Group.OrganizationID,
		AvatarURL:               row.Group.AvatarURL,
		Members:                 ReducedUsersFromGroupMembers(members),
		TotalMemberCount:        totalMemberCount,
		QuotaAllowance:          int(row.Group.QuotaAllowance),
		Source:                  nicloudsdk.GroupSource(row.Group.Source),
		OrganizationName:        row.OrganizationName,
		OrganizationDisplayName: row.OrganizationDisplayName,
	}
}

func TemplateInsightsParameters(parameterRows []database.GetTemplateParameterInsightsRow) ([]nicloudsdk.TemplateParameterUsage, error) {
	// Use a stable sort, similarly to how we would sort in the query, note that
	// we don't sort in the query because order varies depending on the table
	// collation.
	//
	// ORDER BY utp.name, utp.type, utp.display_name, utp.description, utp.options, wbp.value
	slices.SortFunc(parameterRows, func(a, b database.GetTemplateParameterInsightsRow) int {
		if a.Name != b.Name {
			return strings.Compare(a.Name, b.Name)
		}
		if a.Type != b.Type {
			return strings.Compare(a.Type, b.Type)
		}
		if a.DisplayName != b.DisplayName {
			return strings.Compare(a.DisplayName, b.DisplayName)
		}
		if a.Description != b.Description {
			return strings.Compare(a.Description, b.Description)
		}
		if string(a.Options) != string(b.Options) {
			return strings.Compare(string(a.Options), string(b.Options))
		}
		return strings.Compare(a.Value, b.Value)
	})

	parametersUsage := []nicloudsdk.TemplateParameterUsage{}
	indexByNum := make(map[int64]int)
	for _, param := range parameterRows {
		if _, ok := indexByNum[param.Num]; !ok {
			var opts []nicloudsdk.TemplateVersionParameterOption
			err := json.Unmarshal(param.Options, &opts)
			if err != nil {
				return nil, err
			}

			plaintextDescription, err := render.PlaintextFromMarkdown(param.Description)
			if err != nil {
				return nil, err
			}

			parametersUsage = append(parametersUsage, nicloudsdk.TemplateParameterUsage{
				TemplateIDs: param.TemplateIDs,
				Name:        param.Name,
				Type:        param.Type,
				DisplayName: param.DisplayName,
				Description: plaintextDescription,
				Options:     opts,
			})
			indexByNum[param.Num] = len(parametersUsage) - 1
		}

		i := indexByNum[param.Num]
		parametersUsage[i].Values = append(parametersUsage[i].Values, nicloudsdk.TemplateParameterValue{
			Value: param.Value,
			Count: param.Count,
		})
	}

	return parametersUsage, nil
}

func templateVersionParameterOptions(rawOptions json.RawMessage) ([]nicloudsdk.TemplateVersionParameterOption, error) {
	var protoOptions []*proto.RichParameterOption
	err := json.Unmarshal(rawOptions, &protoOptions)
	if err != nil {
		return nil, err
	}

	options := make([]nicloudsdk.TemplateVersionParameterOption, 0)
	for _, option := range protoOptions {
		options = append(options, nicloudsdk.TemplateVersionParameterOption{
			Name:        option.Name,
			Description: option.Description,
			Value:       option.Value,
			Icon:        option.Icon,
		})
	}
	return options, nil
}

func TemplateVersionParameterOptionFromPreview(option *previewtypes.ParameterOption) nicloudsdk.TemplateVersionParameterOption {
	return nicloudsdk.TemplateVersionParameterOption{
		Name:        option.Name,
		Description: option.Description,
		Value:       option.Value.AsString(),
		Icon:        option.Icon,
	}
}

func OAuth2ProviderApp(accessURL *url.URL, dbApp database.OAuth2ProviderApp) nicloudsdk.OAuth2ProviderApp {
	return nicloudsdk.OAuth2ProviderApp{
		ID:          dbApp.ID,
		Name:        dbApp.Name,
		CallbackURL: dbApp.CallbackURL,
		Icon:        dbApp.Icon,
		Endpoints: nicloudsdk.OAuth2AppEndpoints{
			Authorization: accessURL.ResolveReference(&url.URL{
				Path: "/oauth2/authorize",
			}).String(),
			Token: accessURL.ResolveReference(&url.URL{
				Path: "/oauth2/tokens",
			}).String(),
			// We do not currently support DeviceAuth.
			DeviceAuth: "",
			TokenRevoke: accessURL.ResolveReference(&url.URL{
				Path: "/oauth2/revoke",
			}).String(),
		},
	}
}

func OAuth2ProviderApps(accessURL *url.URL, dbApps []database.OAuth2ProviderApp) []nicloudsdk.OAuth2ProviderApp {
	return slice.List(dbApps, func(dbApp database.OAuth2ProviderApp) nicloudsdk.OAuth2ProviderApp {
		return OAuth2ProviderApp(accessURL, dbApp)
	})
}

func convertDisplayApps(apps []database.DisplayApp) []nicloudsdk.DisplayApp {
	dapps := make([]nicloudsdk.DisplayApp, 0, len(apps))
	for _, app := range apps {
		switch nicloudsdk.DisplayApp(app) {
		case nicloudsdk.DisplayAppVSCodeDesktop, nicloudsdk.DisplayAppVSCodeInsiders, nicloudsdk.DisplayAppPortForward, nicloudsdk.DisplayAppWebTerminal, nicloudsdk.DisplayAppSSH:
			dapps = append(dapps, nicloudsdk.DisplayApp(app))
		}
	}

	return dapps
}

func WorkspaceAgentEnvironment(workspaceAgent database.WorkspaceAgent) (map[string]string, error) {
	var envs map[string]string
	if workspaceAgent.EnvironmentVariables.Valid {
		err := json.Unmarshal(workspaceAgent.EnvironmentVariables.RawMessage, &envs)
		if err != nil {
			return nil, xerrors.Errorf("unmarshal environment variables: %w", err)
		}
	}

	return envs, nil
}

func WorkspaceAgent(derpMap *tailcfg.DERPMap, coordinator tailnet.Coordinator,
	dbAgent database.WorkspaceAgent, apps []nicloudsdk.WorkspaceApp, scripts []nicloudsdk.WorkspaceAgentScript, logSources []nicloudsdk.WorkspaceAgentLogSource,
	agentInactiveDisconnectTimeout time.Duration, agentFallbackTroubleshootingURL string,
) (nicloudsdk.WorkspaceAgent, error) {
	envs, err := WorkspaceAgentEnvironment(dbAgent)
	if err != nil {
		return nicloudsdk.WorkspaceAgent{}, err
	}
	troubleshootingURL := agentFallbackTroubleshootingURL
	if dbAgent.TroubleshootingURL != "" {
		troubleshootingURL = dbAgent.TroubleshootingURL
	}
	subsystems := make([]nicloudsdk.AgentSubsystem, len(dbAgent.Subsystems))
	for i, subsystem := range dbAgent.Subsystems {
		subsystems[i] = nicloudsdk.AgentSubsystem(subsystem)
	}

	legacyStartupScriptBehavior := nicloudsdk.WorkspaceAgentStartupScriptBehaviorNonBlocking
	for _, script := range scripts {
		if !script.RunOnStart {
			continue
		}
		if !script.StartBlocksLogin {
			continue
		}
		legacyStartupScriptBehavior = nicloudsdk.WorkspaceAgentStartupScriptBehaviorBlocking
	}

	workspaceAgent := nicloudsdk.WorkspaceAgent{
		ID:                       dbAgent.ID,
		ParentID:                 dbAgent.ParentID,
		CreatedAt:                dbAgent.CreatedAt,
		UpdatedAt:                dbAgent.UpdatedAt,
		ResourceID:               dbAgent.ResourceID,
		InstanceID:               dbAgent.AuthInstanceID.String,
		Name:                     dbAgent.Name,
		Architecture:             dbAgent.Architecture,
		OperatingSystem:          dbAgent.OperatingSystem,
		Scripts:                  scripts,
		StartupScriptBehavior:    legacyStartupScriptBehavior,
		LogsLength:               dbAgent.LogsLength,
		LogsOverflowed:           dbAgent.LogsOverflowed,
		LogSources:               logSources,
		Version:                  dbAgent.Version,
		APIVersion:               dbAgent.APIVersion,
		EnvironmentVariables:     envs,
		Directory:                dbAgent.Directory,
		ExpandedDirectory:        dbAgent.ExpandedDirectory,
		Apps:                     apps,
		ConnectionTimeoutSeconds: dbAgent.ConnectionTimeoutSeconds,
		TroubleshootingURL:       troubleshootingURL,
		LifecycleState:           nicloudsdk.WorkspaceAgentLifecycle(dbAgent.LifecycleState),
		Subsystems:               subsystems,
		DisplayApps:              convertDisplayApps(dbAgent.DisplayApps),
	}
	node := coordinator.Node(dbAgent.ID)
	if node != nil {
		workspaceAgent.DERPLatency = map[string]nicloudsdk.DERPRegion{}
		for rawRegion, latency := range node.DERPLatency {
			regionParts := strings.SplitN(rawRegion, "-", 2)
			regionID, err := strconv.Atoi(regionParts[0])
			if err != nil {
				return nicloudsdk.WorkspaceAgent{}, xerrors.Errorf("convert derp region id %q: %w", rawRegion, err)
			}
			region, found := derpMap.Regions[regionID]
			if !found {
				// It's possible that a workspace agent is using an old DERPMap
				// and reports regions that do not exist. If that's the case,
				// report the region as unknown!
				region = &tailcfg.DERPRegion{
					RegionID:   regionID,
					RegionName: fmt.Sprintf("Unnamed %d", regionID),
				}
			}
			workspaceAgent.DERPLatency[region.RegionName] = nicloudsdk.DERPRegion{
				Preferred:           node.PreferredDERP == regionID,
				LatencyMilliseconds: latency * 1000,
			}
		}
	}

	status := dbAgent.Status(dbtime.Now(), agentInactiveDisconnectTimeout)
	workspaceAgent.Status = nicloudsdk.WorkspaceAgentStatus(status.Status)
	workspaceAgent.FirstConnectedAt = status.FirstConnectedAt
	workspaceAgent.LastConnectedAt = status.LastConnectedAt
	workspaceAgent.DisconnectedAt = status.DisconnectedAt

	if dbAgent.StartedAt.Valid {
		workspaceAgent.StartedAt = &dbAgent.StartedAt.Time
	}
	if dbAgent.ReadyAt.Valid {
		workspaceAgent.ReadyAt = &dbAgent.ReadyAt.Time
	}

	switch {
	case workspaceAgent.Status != nicloudsdk.WorkspaceAgentConnected && workspaceAgent.LifecycleState == nicloudsdk.WorkspaceAgentLifecycleOff:
		workspaceAgent.Health.Reason = "agent is not running"
	case workspaceAgent.Status == nicloudsdk.WorkspaceAgentConnecting:
		// Note: the case above catches connecting+off as "not running".
		// This case handles connecting agents with a non-off lifecycle
		// (e.g. "created" or "starting"), where the agent binary has
		// not yet established a connection to nicloud.
		workspaceAgent.Health.Reason = "agent has not yet connected"
	case workspaceAgent.Status == nicloudsdk.WorkspaceAgentTimeout:
		workspaceAgent.Health.Reason = "agent is taking too long to connect"
	case workspaceAgent.Status == nicloudsdk.WorkspaceAgentDisconnected:
		workspaceAgent.Health.Reason = "agent has lost connection"
	// Note: We could also handle nicloudsdk.WorkspaceAgentLifecycleStartTimeout
	// here, but it's more of a soft issue, so we don't want to mark the agent
	// as unhealthy.
	case workspaceAgent.LifecycleState == nicloudsdk.WorkspaceAgentLifecycleStartError:
		workspaceAgent.Health.Reason = "agent startup script exited with an error"
	case workspaceAgent.LifecycleState.ShuttingDown():
		workspaceAgent.Health.Reason = "agent is shutting down"
	default:
		workspaceAgent.Health.Healthy = true
	}

	return workspaceAgent, nil
}

func AppSubdomain(dbApp database.WorkspaceApp, agentName, workspaceName, ownerName string) string {
	if !dbApp.Subdomain || agentName == "" || ownerName == "" || workspaceName == "" {
		return ""
	}

	appSlug := dbApp.Slug
	if appSlug == "" {
		appSlug = dbApp.DisplayName
	}

	// Agent name is optional when app slug is present
	normalizedAgentName := agentName
	if !appurl.PortRegex.MatchString(appSlug) {
		normalizedAgentName = ""
	}

	return appurl.ApplicationURL{
		// We never generate URLs with a prefix. We only allow prefixes when
		// parsing URLs from the hostname. Users that want this feature can
		// write out their own URLs.
		Prefix:        "",
		AppSlugOrPort: appSlug,
		AgentName:     normalizedAgentName,
		WorkspaceName: workspaceName,
		Username:      ownerName,
	}.String()
}

func Apps(dbApps []database.WorkspaceApp, statuses []database.WorkspaceAppStatus, agent database.WorkspaceAgent, ownerName string, workspace database.WorkspaceTable) []nicloudsdk.WorkspaceApp {
	sort.Slice(dbApps, func(i, j int) bool {
		if dbApps[i].DisplayOrder != dbApps[j].DisplayOrder {
			return dbApps[i].DisplayOrder < dbApps[j].DisplayOrder
		}
		if dbApps[i].DisplayName != dbApps[j].DisplayName {
			return dbApps[i].DisplayName < dbApps[j].DisplayName
		}
		return dbApps[i].Slug < dbApps[j].Slug
	})

	statusesByAppID := map[uuid.UUID][]database.WorkspaceAppStatus{}
	for _, status := range statuses {
		statusesByAppID[status.AppID] = append(statusesByAppID[status.AppID], status)
	}

	apps := make([]nicloudsdk.WorkspaceApp, 0)
	for _, dbApp := range dbApps {
		statuses := statusesByAppID[dbApp.ID]
		apps = append(apps, nicloudsdk.WorkspaceApp{
			ID:            dbApp.ID,
			URL:           dbApp.Url.String,
			External:      dbApp.External,
			Slug:          dbApp.Slug,
			DisplayName:   dbApp.DisplayName,
			Command:       dbApp.Command.String,
			Icon:          dbApp.Icon,
			Subdomain:     dbApp.Subdomain,
			SubdomainName: AppSubdomain(dbApp, agent.Name, workspace.Name, ownerName),
			SharingLevel:  nicloudsdk.WorkspaceAppSharingLevel(dbApp.SharingLevel),
			Healthcheck: nicloudsdk.Healthcheck{
				URL:       dbApp.HealthcheckUrl,
				Interval:  dbApp.HealthcheckInterval,
				Threshold: dbApp.HealthcheckThreshold,
			},
			Health:   nicloudsdk.WorkspaceAppHealth(dbApp.Health),
			Group:    dbApp.DisplayGroup.String,
			Hidden:   dbApp.Hidden,
			OpenIn:   nicloudsdk.WorkspaceAppOpenIn(dbApp.OpenIn),
			Tooltip:  dbApp.Tooltip,
			Statuses: WorkspaceAppStatuses(statuses),
		})
	}
	return apps
}

func WorkspaceAppStatuses(statuses []database.WorkspaceAppStatus) []nicloudsdk.WorkspaceAppStatus {
	return slice.List(statuses, WorkspaceAppStatus)
}

func WorkspaceAppStatus(status database.WorkspaceAppStatus) nicloudsdk.WorkspaceAppStatus {
	return nicloudsdk.WorkspaceAppStatus{
		ID:          status.ID,
		CreatedAt:   status.CreatedAt,
		WorkspaceID: status.WorkspaceID,
		AgentID:     status.AgentID,
		AppID:       status.AppID,
		URI:         status.Uri.String,
		Message:     status.Message,
		State:       nicloudsdk.WorkspaceAppStatusState(status.State),
	}
}

func ProvisionerJobLog(log database.ProvisionerJobLog) nicloudsdk.ProvisionerJobLog {
	return nicloudsdk.ProvisionerJobLog{
		ID:        log.ID,
		CreatedAt: log.CreatedAt,
		Source:    nicloudsdk.LogSource(log.Source),
		Level:     nicloudsdk.LogLevel(log.Level),
		Stage:     log.Stage,
		Output:    log.Output,
	}
}

func WorkspaceAgentLog(log database.WorkspaceAgentLog) nicloudsdk.WorkspaceAgentLog {
	return nicloudsdk.WorkspaceAgentLog{
		ID:        log.ID,
		CreatedAt: log.CreatedAt,
		Output:    log.Output,
		Level:     nicloudsdk.LogLevel(log.Level),
		SourceID:  log.LogSourceID,
	}
}

func WorkspaceAgentScript(dbScript database.GetWorkspaceAgentScriptsByAgentIDsRow) nicloudsdk.WorkspaceAgentScript {
	script := nicloudsdk.WorkspaceAgentScript{
		ID:               dbScript.ID,
		LogPath:          dbScript.LogPath,
		LogSourceID:      dbScript.LogSourceID,
		Script:           dbScript.Script,
		Cron:             dbScript.Cron,
		RunOnStart:       dbScript.RunOnStart,
		RunOnStop:        dbScript.RunOnStop,
		StartBlocksLogin: dbScript.StartBlocksLogin,
		Timeout:          time.Duration(dbScript.TimeoutSeconds) * time.Second,
		DisplayName:      dbScript.DisplayName,
		ExitCode:         nullInt32Ptr(dbScript.ExitCode),
	}
	if dbScript.Status.Valid {
		status := nicloudsdk.WorkspaceAgentScriptStatus(dbScript.Status.WorkspaceAgentScriptTimingStatus)
		script.Status = &status
	}
	return script
}

func ProvisionerDaemon(dbDaemon database.ProvisionerDaemon) nicloudsdk.ProvisionerDaemon {
	result := nicloudsdk.ProvisionerDaemon{
		ID:             dbDaemon.ID,
		OrganizationID: dbDaemon.OrganizationID,
		CreatedAt:      dbDaemon.CreatedAt,
		LastSeenAt:     nicloudsdk.NullTime{NullTime: dbDaemon.LastSeenAt},
		Name:           dbDaemon.Name,
		Tags:           dbDaemon.Tags,
		Version:        dbDaemon.Version,
		APIVersion:     dbDaemon.APIVersion,
		KeyID:          dbDaemon.KeyID,
	}
	for _, provisionerType := range dbDaemon.Provisioners {
		result.Provisioners = append(result.Provisioners, nicloudsdk.ProvisionerType(provisionerType))
	}
	return result
}

func RecentProvisionerDaemons(now time.Time, staleInterval time.Duration, daemons []database.ProvisionerDaemon) []nicloudsdk.ProvisionerDaemon {
	results := []nicloudsdk.ProvisionerDaemon{}

	for _, daemon := range daemons {
		// Daemon never connected, skip.
		if !daemon.LastSeenAt.Valid {
			continue
		}
		// Daemon has gone away, skip.
		if now.Sub(daemon.LastSeenAt.Time) > staleInterval {
			continue
		}

		results = append(results, ProvisionerDaemon(daemon))
	}

	// Ensure stable order for display and for tests
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

func SlimRole(role rbac.Role) nicloudsdk.SlimRole {
	orgID := ""
	if role.Identifier.OrganizationID != uuid.Nil {
		orgID = role.Identifier.OrganizationID.String()
	}

	return nicloudsdk.SlimRole{
		DisplayName:    role.DisplayName,
		Name:           role.Identifier.Name,
		OrganizationID: orgID,
	}
}

func SlimRolesFromNames(names []string) []nicloudsdk.SlimRole {
	convertedRoles := make([]nicloudsdk.SlimRole, 0, len(names))

	for _, name := range names {
		convertedRoles = append(convertedRoles, SlimRoleFromName(name))
	}

	return convertedRoles
}

func SlimRoleFromName(name string) nicloudsdk.SlimRole {
	rbacRole, err := rbac.RoleByName(rbac.RoleIdentifier{Name: name})
	var convertedRole nicloudsdk.SlimRole
	if err == nil {
		convertedRole = SlimRole(rbacRole)
	} else {
		convertedRole = nicloudsdk.SlimRole{Name: name}
	}
	return convertedRole
}

func RBACRole(role rbac.Role) nicloudsdk.Role {
	slim := SlimRole(role)

	orgPerms := role.ByOrgID[slim.OrganizationID]
	return nicloudsdk.Role{
		Name:                          slim.Name,
		OrganizationID:                slim.OrganizationID,
		DisplayName:                   slim.DisplayName,
		SitePermissions:               slice.List(role.Site, RBACPermission),
		UserPermissions:               slice.List(role.User, RBACPermission),
		OrganizationPermissions:       slice.List(orgPerms.Org, RBACPermission),
		OrganizationMemberPermissions: slice.List(orgPerms.Member, RBACPermission),
	}
}

func Role(role database.CustomRole) nicloudsdk.Role {
	orgID := ""
	if role.OrganizationID.UUID != uuid.Nil {
		orgID = role.OrganizationID.UUID.String()
	}

	return nicloudsdk.Role{
		Name:                    role.Name,
		OrganizationID:          orgID,
		DisplayName:             role.DisplayName,
		SitePermissions:         slice.List(role.SitePermissions, Permission),
		UserPermissions:         slice.List(role.UserPermissions, Permission),
		OrganizationPermissions: slice.List(role.OrgPermissions, Permission),
	}
}

func Permission(permission database.CustomRolePermission) nicloudsdk.Permission {
	return nicloudsdk.Permission{
		Negate:       permission.Negate,
		ResourceType: nicloudsdk.RBACResource(permission.ResourceType),
		Action:       nicloudsdk.RBACAction(permission.Action),
	}
}

func RBACPermission(permission rbac.Permission) nicloudsdk.Permission {
	return nicloudsdk.Permission{
		Negate:       permission.Negate,
		ResourceType: nicloudsdk.RBACResource(permission.ResourceType),
		Action:       nicloudsdk.RBACAction(permission.Action),
	}
}

func Organization(organization database.Organization) nicloudsdk.Organization {
	return nicloudsdk.Organization{
		MinimalOrganization: nicloudsdk.MinimalOrganization{
			ID:          organization.ID,
			Name:        organization.Name,
			DisplayName: organization.DisplayName,
			Icon:        organization.Icon,
		},
		Description: organization.Description,
		CreatedAt:   organization.CreatedAt,
		UpdatedAt:   organization.UpdatedAt,
		IsDefault:   organization.IsDefault,
	}
}

func CryptoKeys(keys []database.CryptoKey) []nicloudsdk.CryptoKey {
	return slice.List(keys, CryptoKey)
}

func CryptoKey(key database.CryptoKey) nicloudsdk.CryptoKey {
	return nicloudsdk.CryptoKey{
		Feature:   nicloudsdk.CryptoKeyFeature(key.Feature),
		Sequence:  key.Sequence,
		StartsAt:  key.StartsAt,
		DeletesAt: key.DeletesAt.Time,
		Secret:    key.Secret.String,
	}
}

func MatchedProvisioners(provisionerDaemons []database.ProvisionerDaemon, now time.Time, staleInterval time.Duration) nicloudsdk.MatchedProvisioners {
	minLastSeenAt := now.Add(-staleInterval)
	mostRecentlySeen := nicloudsdk.NullTime{}
	var matched nicloudsdk.MatchedProvisioners
	for _, provisioner := range provisionerDaemons {
		if !provisioner.LastSeenAt.Valid {
			continue
		}
		matched.Count++
		if provisioner.LastSeenAt.Time.After(minLastSeenAt) {
			matched.Available++
		}
		if provisioner.LastSeenAt.Time.After(mostRecentlySeen.Time) {
			matched.MostRecentlySeen.Valid = true
			matched.MostRecentlySeen.Time = provisioner.LastSeenAt.Time
		}
	}
	return matched
}

func TemplateRoleActions(role nicloudsdk.TemplateRole) []policy.Action {
	switch role {
	case nicloudsdk.TemplateRoleAdmin:
		return []policy.Action{policy.WildcardSymbol}
	case nicloudsdk.TemplateRoleUse:
		return []policy.Action{policy.ActionRead, policy.ActionUse}
	}
	return []policy.Action{}
}

func WorkspaceRoleActions(role nicloudsdk.WorkspaceRole) []policy.Action {
	switch role {
	case nicloudsdk.WorkspaceRoleAdmin:
		return slice.Omit(
			// Small note: This intentionally includes "create" because it's sort of
			// double purposed as "can edit ACL". That's maybe a bit "incorrect", but
			// it's what templates do already and we're copying that implementation.
			rbac.ResourceWorkspace.AvailableActions(),
			// Don't let anyone delete something they can't recreate.
			policy.ActionDelete,
		)
	case nicloudsdk.WorkspaceRoleUse:
		return []policy.Action{
			policy.ActionApplicationConnect,
			policy.ActionRead,
			policy.ActionSSH,
			policy.ActionWorkspaceStart,
			policy.ActionWorkspaceStop,
		}
	}
	return []policy.Action{}
}

func ChatRoleActions(role nicloudsdk.ChatRole) []policy.Action {
	if role == nicloudsdk.ChatRoleRead {
		return []policy.Action{policy.ActionRead}
	}
	return []policy.Action{}
}

func ConnectionLogConnectionTypeFromAgentProtoConnectionType(typ agentproto.Connection_Type) (database.ConnectionType, error) {
	switch typ {
	case agentproto.Connection_SSH:
		return database.ConnectionTypeSsh, nil
	case agentproto.Connection_JETBRAINS:
		return database.ConnectionTypeJetbrains, nil
	case agentproto.Connection_VSCODE:
		return database.ConnectionTypeVscode, nil
	case agentproto.Connection_RECONNECTING_PTY:
		return database.ConnectionTypeReconnectingPty, nil
	default:
		// Also Connection_TYPE_UNSPECIFIED, no mapping.
		return "", xerrors.Errorf("unknown agent connection type %q", typ)
	}
}

func ConnectionLogStatusFromAgentProtoConnectionAction(action agentproto.Connection_Action) (database.ConnectionStatus, error) {
	switch action {
	case agentproto.Connection_CONNECT:
		return database.ConnectionStatusConnected, nil
	case agentproto.Connection_DISCONNECT:
		return database.ConnectionStatusDisconnected, nil
	default:
		// Also Connection_ACTION_UNSPECIFIED, no mapping.
		return "", xerrors.Errorf("unknown agent connection action %q", action)
	}
}

func PreviewParameter(param previewtypes.Parameter) nicloudsdk.PreviewParameter {
	return nicloudsdk.PreviewParameter{
		PreviewParameterData: nicloudsdk.PreviewParameterData{
			Name:        param.Name,
			DisplayName: param.DisplayName,
			Description: param.Description,
			Type:        nicloudsdk.OptionType(param.Type),
			FormType:    nicloudsdk.ParameterFormType(param.FormType),
			Styling: nicloudsdk.PreviewParameterStyling{
				Placeholder: param.Styling.Placeholder,
				Disabled:    param.Styling.Disabled,
				Label:       param.Styling.Label,
				MaskInput:   param.Styling.MaskInput,
			},
			Mutable:      param.Mutable,
			DefaultValue: PreviewHCLString(param.DefaultValue),
			Icon:         param.Icon,
			Options:      slice.List(param.Options, PreviewParameterOption),
			Validations:  slice.List(param.Validations, PreviewParameterValidation),
			Required:     param.Required,
			Order:        param.Order,
			Ephemeral:    param.Ephemeral,
		},
		Value:       PreviewHCLString(param.Value),
		Diagnostics: PreviewDiagnostics(param.Diagnostics),
	}
}

func HCLDiagnostics(d hcl.Diagnostics) []nicloudsdk.FriendlyDiagnostic {
	return PreviewDiagnostics(previewtypes.Diagnostics(d))
}

func PreviewDiagnostics(d previewtypes.Diagnostics) []nicloudsdk.FriendlyDiagnostic {
	f := d.FriendlyDiagnostics()
	return slice.List(f, func(f previewtypes.FriendlyDiagnostic) nicloudsdk.FriendlyDiagnostic {
		return nicloudsdk.FriendlyDiagnostic{
			Severity: nicloudsdk.DiagnosticSeverityString(f.Severity),
			Summary:  f.Summary,
			Detail:   f.Detail,
			Extra: nicloudsdk.DiagnosticExtra{
				Code: f.Extra.Code,
			},
		}
	})
}

func PreviewHCLString(h previewtypes.HCLString) nicloudsdk.NullHCLString {
	n := h.NullHCLString()
	return nicloudsdk.NullHCLString{
		Value: n.Value,
		Valid: n.Valid,
	}
}

func PreviewParameterOption(o *previewtypes.ParameterOption) nicloudsdk.PreviewParameterOption {
	if o == nil {
		// This should never be sent
		return nicloudsdk.PreviewParameterOption{}
	}
	return nicloudsdk.PreviewParameterOption{
		Name:        o.Name,
		Description: o.Description,
		Value:       PreviewHCLString(o.Value),
		Icon:        o.Icon,
	}
}

func PreviewParameterValidation(v *previewtypes.ParameterValidation) nicloudsdk.PreviewParameterValidation {
	if v == nil {
		// This should never be sent
		return nicloudsdk.PreviewParameterValidation{}
	}
	return nicloudsdk.PreviewParameterValidation{
		Error:     v.Error,
		Regex:     v.Regex,
		Min:       v.Min,
		Max:       v.Max,
		Monotonic: v.Monotonic,
	}
}

func AIBridgeInterception(interception database.AIBridgeInterception, initiator database.VisibleUser, tokenUsages []database.AIBridgeTokenUsage, userPrompts []database.AIBridgeUserPrompt, toolUsages []database.AIBridgeToolUsage) nicloudsdk.AIBridgeInterception {
	sdkTokenUsages := slice.List(tokenUsages, AIBridgeTokenUsage)
	sort.Slice(sdkTokenUsages, func(i, j int) bool {
		// created_at ASC
		return sdkTokenUsages[i].CreatedAt.Before(sdkTokenUsages[j].CreatedAt)
	})
	sdkUserPrompts := slice.List(userPrompts, AIBridgeUserPrompt)
	sort.Slice(sdkUserPrompts, func(i, j int) bool {
		// created_at ASC
		return sdkUserPrompts[i].CreatedAt.Before(sdkUserPrompts[j].CreatedAt)
	})
	sdkToolUsages := slice.List(toolUsages, AIBridgeToolUsage)
	sort.Slice(sdkToolUsages, func(i, j int) bool {
		// created_at ASC
		return sdkToolUsages[i].CreatedAt.Before(sdkToolUsages[j].CreatedAt)
	})
	intc := nicloudsdk.AIBridgeInterception{
		ID:           interception.ID,
		Initiator:    MinimalUserFromVisibleUser(initiator),
		Provider:     interception.Provider,
		ProviderName: interception.ProviderName,
		Model:        interception.Model,
		Metadata:     jsonOrEmptyMap(interception.Metadata),
		StartedAt:    interception.StartedAt,
		TokenUsages:  sdkTokenUsages,
		UserPrompts:  sdkUserPrompts,
		ToolUsages:   sdkToolUsages,
	}
	if interception.APIKeyID.Valid {
		intc.APIKeyID = &interception.APIKeyID.String
	}
	if interception.EndedAt.Valid {
		intc.EndedAt = &interception.EndedAt.Time
	}
	if interception.Client.Valid {
		intc.Client = &interception.Client.String
	}
	return intc
}

func AIBridgeSession(row database.ListAIBridgeSessionsRow) nicloudsdk.AIBridgeSession {
	session := nicloudsdk.AIBridgeSession{
		ID: row.SessionID,
		Initiator: MinimalUserFromVisibleUser(database.VisibleUser{
			ID:        row.UserID,
			Username:  row.UserUsername,
			Name:      row.UserName,
			AvatarURL: row.UserAvatarUrl,
		}),
		Providers:    row.Providers,
		Models:       row.Models,
		Metadata:     jsonOrEmptyMap(pqtype.NullRawMessage{RawMessage: row.Metadata, Valid: len(row.Metadata) > 0}),
		StartedAt:    row.StartedAt,
		Threads:      row.Threads,
		LastActiveAt: row.LastActiveAt,
		TokenUsageSummary: nicloudsdk.AIBridgeSessionTokenUsageSummary{
			InputTokens:           row.InputTokens,
			OutputTokens:          row.OutputTokens,
			CacheReadInputTokens:  row.CacheReadInputTokens,
			CacheWriteInputTokens: row.CacheWriteInputTokens,
		},
	}
	// Ensure non-nil slices for JSON serialization.
	if session.Providers == nil {
		session.Providers = []string{}
	}
	if session.Models == nil {
		session.Models = []string{}
	}
	if row.Client != "" {
		session.Client = &row.Client
	}
	if !row.EndedAt.IsZero() {
		session.EndedAt = &row.EndedAt
	}
	if row.LastPrompt != "" {
		session.LastPrompt = &row.LastPrompt
	}
	return session
}

func AIBridgeTokenUsage(usage database.AIBridgeTokenUsage) nicloudsdk.AIBridgeTokenUsage {
	return nicloudsdk.AIBridgeTokenUsage{
		ID:                    usage.ID,
		InterceptionID:        usage.InterceptionID,
		ProviderResponseID:    usage.ProviderResponseID,
		InputTokens:           usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		CacheReadInputTokens:  usage.CacheReadInputTokens,
		CacheWriteInputTokens: usage.CacheWriteInputTokens,
		Metadata:              jsonOrEmptyMap(usage.Metadata),
		CreatedAt:             usage.CreatedAt,
	}
}

func AIBridgeUserPrompt(prompt database.AIBridgeUserPrompt) nicloudsdk.AIBridgeUserPrompt {
	return nicloudsdk.AIBridgeUserPrompt{
		ID:                 prompt.ID,
		InterceptionID:     prompt.InterceptionID,
		ProviderResponseID: prompt.ProviderResponseID,
		Prompt:             prompt.Prompt,
		Metadata:           jsonOrEmptyMap(prompt.Metadata),
		CreatedAt:          prompt.CreatedAt,
	}
}

func AIBridgeToolUsage(usage database.AIBridgeToolUsage) nicloudsdk.AIBridgeToolUsage {
	return nicloudsdk.AIBridgeToolUsage{
		ID:                 usage.ID,
		InterceptionID:     usage.InterceptionID,
		ProviderResponseID: usage.ProviderResponseID,
		ServerURL:          usage.ServerUrl.String,
		Tool:               usage.Tool,
		Input:              usage.Input,
		Injected:           usage.Injected,
		InvocationError:    usage.InvocationError.String,
		Metadata:           jsonOrEmptyMap(usage.Metadata),
		CreatedAt:          usage.CreatedAt,
	}
}

// AIBridgeSessionThreads converts session metadata and thread interceptions
// into the threads response. It groups interceptions into threads, builds
// agentic actions from tool usages and model thoughts, and aggregates
// token usage with metadata.
func AIBridgeSessionThreads(
	session database.ListAIBridgeSessionsRow,
	interceptions []database.ListAIBridgeSessionThreadsRow,
	tokenUsages []database.AIBridgeTokenUsage,
	toolUsages []database.AIBridgeToolUsage,
	userPrompts []database.AIBridgeUserPrompt,
	modelThoughts []database.AIBridgeModelThought,
) nicloudsdk.AIBridgeSessionThreadsResponse {
	// Index subresources by interception ID.
	tokensByInterception := make(map[uuid.UUID][]database.AIBridgeTokenUsage, len(interceptions))
	for _, tu := range tokenUsages {
		tokensByInterception[tu.InterceptionID] = append(tokensByInterception[tu.InterceptionID], tu)
	}
	toolsByInterception := make(map[uuid.UUID][]database.AIBridgeToolUsage, len(interceptions))
	for _, tu := range toolUsages {
		toolsByInterception[tu.InterceptionID] = append(toolsByInterception[tu.InterceptionID], tu)
	}
	promptsByInterception := make(map[uuid.UUID][]database.AIBridgeUserPrompt, len(interceptions))
	for _, up := range userPrompts {
		promptsByInterception[up.InterceptionID] = append(promptsByInterception[up.InterceptionID], up)
	}
	thoughtsByInterception := make(map[uuid.UUID][]database.AIBridgeModelThought, len(interceptions))
	for _, mt := range modelThoughts {
		thoughtsByInterception[mt.InterceptionID] = append(thoughtsByInterception[mt.InterceptionID], mt)
	}

	// Group interceptions by thread_id, preserving the order returned by the
	// SQL query.
	interceptionsByThread := make(map[uuid.UUID][]database.AIBridgeInterception, len(interceptions))
	var threadIDs []uuid.UUID
	for _, row := range interceptions {
		if _, ok := interceptionsByThread[row.ThreadID]; !ok {
			threadIDs = append(threadIDs, row.ThreadID)
		}
		interceptionsByThread[row.ThreadID] = append(interceptionsByThread[row.ThreadID], row.AIBridgeInterception)
	}

	// Build threads and track page time bounds.
	threads := make([]nicloudsdk.AIBridgeThread, 0, len(threadIDs))
	var pageStartedAt, pageEndedAt *time.Time
	for _, threadID := range threadIDs {
		intcs := interceptionsByThread[threadID]
		thread := buildAIBridgeThread(threadID, intcs, tokensByInterception, toolsByInterception, promptsByInterception, thoughtsByInterception)
		for _, intc := range intcs {
			if pageStartedAt == nil || intc.StartedAt.Before(*pageStartedAt) {
				t := intc.StartedAt
				pageStartedAt = &t
			}
			if intc.EndedAt.Valid {
				if pageEndedAt == nil || intc.EndedAt.Time.After(*pageEndedAt) {
					t := intc.EndedAt.Time
					pageEndedAt = &t
				}
			}
		}
		threads = append(threads, thread)
	}

	// Aggregate session-level token usage metadata from all token
	// usages in the session (not just the page).
	sessionTokenMeta := aggregateTokenMetadata(tokenUsages)

	resp := nicloudsdk.AIBridgeSessionThreadsResponse{
		ID: session.SessionID,
		Initiator: MinimalUserFromVisibleUser(database.VisibleUser{
			ID:        session.UserID,
			Username:  session.UserUsername,
			Name:      session.UserName,
			AvatarURL: session.UserAvatarUrl,
		}),
		Providers:     session.Providers,
		Models:        session.Models,
		Metadata:      jsonOrEmptyMap(pqtype.NullRawMessage{RawMessage: session.Metadata, Valid: len(session.Metadata) > 0}),
		StartedAt:     session.StartedAt,
		PageStartedAt: pageStartedAt,
		PageEndedAt:   pageEndedAt,
		TokenUsageSummary: nicloudsdk.AIBridgeSessionThreadsTokenUsage{
			InputTokens:           session.InputTokens,
			OutputTokens:          session.OutputTokens,
			CacheReadInputTokens:  session.CacheReadInputTokens,
			CacheWriteInputTokens: session.CacheWriteInputTokens,
			Metadata:              sessionTokenMeta,
		},
		Threads: threads,
	}
	if resp.Providers == nil {
		resp.Providers = []string{}
	}
	if resp.Models == nil {
		resp.Models = []string{}
	}
	if session.Client != "" {
		resp.Client = &session.Client
	}
	if !session.EndedAt.IsZero() {
		resp.EndedAt = &session.EndedAt
	}
	return resp
}

func buildAIBridgeThread(
	threadID uuid.UUID,
	interceptions []database.AIBridgeInterception,
	tokensByInterception map[uuid.UUID][]database.AIBridgeTokenUsage,
	toolsByInterception map[uuid.UUID][]database.AIBridgeToolUsage,
	promptsByInterception map[uuid.UUID][]database.AIBridgeUserPrompt,
	thoughtsByInterception map[uuid.UUID][]database.AIBridgeModelThought,
) nicloudsdk.AIBridgeThread {
	// Find the root interception (where id == threadID) to get the
	// thread prompt and model.
	var rootIntc *database.AIBridgeInterception
	for i := range interceptions {
		if interceptions[i].ID == threadID {
			rootIntc = &interceptions[i]
			break
		}
	}
	// Fallback to first interception if root not found.
	if rootIntc == nil && len(interceptions) > 0 {
		rootIntc = &interceptions[0]
	}

	thread := nicloudsdk.AIBridgeThread{
		ID: threadID,
	}
	if rootIntc != nil {
		thread.Model = rootIntc.Model
		thread.Provider = rootIntc.Provider
		thread.CredentialKind = string(rootIntc.CredentialKind)
		thread.CredentialHint = sanitizeCredentialHint(rootIntc.CredentialHint)
		// Get first user prompt from root interception.
		// A thread can only have one prompt, by definition, since we currently
		// only store the last prompt observed in an interception.
		if prompts := promptsByInterception[rootIntc.ID]; len(prompts) > 0 {
			thread.Prompt = &prompts[0].Prompt
		}
	}

	// Compute thread time bounds from interceptions.
	for _, intc := range interceptions {
		if thread.StartedAt.IsZero() || intc.StartedAt.Before(thread.StartedAt) {
			thread.StartedAt = intc.StartedAt
		}
		if intc.EndedAt.Valid {
			if thread.EndedAt == nil || intc.EndedAt.Time.After(*thread.EndedAt) {
				t := intc.EndedAt.Time
				thread.EndedAt = &t
			}
		}
	}

	// Build agentic actions grouped by interception. Each interception that
	// has tool calls produces one action with all its tool calls, thinking
	// blocks, and token usage.
	var actions []nicloudsdk.AIBridgeAgenticAction
	for _, intc := range interceptions {
		tools := toolsByInterception[intc.ID]
		if len(tools) == 0 {
			continue
		}

		// Thinking blocks for this interception.
		thoughts := thoughtsByInterception[intc.ID]
		thinking := make([]nicloudsdk.AIBridgeModelThought, 0, len(thoughts))
		for _, mt := range thoughts {
			thinking = append(thinking, nicloudsdk.AIBridgeModelThought{
				Text: mt.Content,
			})
		}

		// Token usage for the interception.
		actionTokenUsage := aggregateTokenUsage(tokensByInterception[intc.ID])

		// Build tool call list.
		toolCalls := make([]nicloudsdk.AIBridgeToolCall, 0, len(tools))
		for _, tu := range tools {
			toolCalls = append(toolCalls, nicloudsdk.AIBridgeToolCall{
				ID:                 tu.ID,
				InterceptionID:     tu.InterceptionID,
				ProviderResponseID: tu.ProviderResponseID,
				ServerURL:          tu.ServerUrl.String,
				Tool:               tu.Tool,
				Injected:           tu.Injected,
				Input:              tu.Input,
				Metadata:           jsonOrEmptyMap(tu.Metadata),
				CreatedAt:          tu.CreatedAt,
			})
		}

		actions = append(actions, nicloudsdk.AIBridgeAgenticAction{
			Model:      intc.Model,
			TokenUsage: actionTokenUsage,
			Thinking:   thinking,
			ToolCalls:  toolCalls,
		})
	}

	if actions == nil {
		// Make an empty slice so we don't serialize `null`.
		actions = make([]nicloudsdk.AIBridgeAgenticAction, 0)
	}

	thread.AgenticActions = actions

	// Aggregate thread-level token usage.
	var threadTokens []database.AIBridgeTokenUsage
	for _, intc := range interceptions {
		threadTokens = append(threadTokens, tokensByInterception[intc.ID]...)
	}
	thread.TokenUsage = aggregateTokenUsage(threadTokens)

	return thread
}

// aggregateTokenUsage sums token usage rows and aggregates metadata.
func aggregateTokenUsage(tokens []database.AIBridgeTokenUsage) nicloudsdk.AIBridgeSessionThreadsTokenUsage {
	var inputTokens, outputTokens, cacheRead, cacheWrite int64
	for _, tu := range tokens {
		inputTokens += tu.InputTokens
		outputTokens += tu.OutputTokens
		cacheRead += tu.CacheReadInputTokens
		cacheWrite += tu.CacheWriteInputTokens
	}
	return nicloudsdk.AIBridgeSessionThreadsTokenUsage{
		InputTokens:           inputTokens,
		OutputTokens:          outputTokens,
		CacheReadInputTokens:  cacheRead,
		CacheWriteInputTokens: cacheWrite,
		Metadata:              aggregateTokenMetadata(tokens),
	}
}

// aggregateTokenMetadata sums all numeric values from the metadata
// JSONB across the given token usage rows by key. Nested objects are
// flattened using dot-notation (e.g. {"cache": {"read_tokens": 10}}
// becomes "cache.read_tokens"). Non-numeric leaves (strings,
// booleans, arrays, nulls) are silently skipped.
func aggregateTokenMetadata(tokens []database.AIBridgeTokenUsage) map[string]any {
	sums := make(map[string]int64)
	for _, tu := range tokens {
		if !tu.Metadata.Valid || len(tu.Metadata.RawMessage) == 0 {
			continue
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(tu.Metadata.RawMessage, &m); err != nil {
			continue
		}
		flattenAndSum(sums, "", m)
	}
	result := make(map[string]any, len(sums))
	for k, v := range sums {
		result[k] = v
	}
	return result
}

// flattenAndSum recursively walks a JSON object and sums all numeric
// leaf values into sums, using dot-separated keys for nested objects.
func flattenAndSum(sums map[string]int64, prefix string, m map[string]json.RawMessage) {
	for k, raw := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		// Try as a number first.
		var n json.Number
		if err := json.Unmarshal(raw, &n); err == nil {
			if v, err := n.Int64(); err == nil {
				sums[key] += v
			}
			continue
		}

		// Try as a nested object.
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(raw, &nested); err == nil {
			flattenAndSum(sums, key, nested)
		}
		// Arrays, strings, booleans, nulls are skipped.
	}
}

func GroupAIBudget(b database.GroupAiBudget) nicloudsdk.GroupAIBudget {
	return nicloudsdk.GroupAIBudget{
		GroupID:          b.GroupID,
		SpendLimitMicros: b.SpendLimitMicros,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
	}
}

func UserAIBudgetOverride(o database.UserAiBudgetOverride) nicloudsdk.UserAIBudgetOverride {
	return nicloudsdk.UserAIBudgetOverride{
		UserID:           o.UserID,
		GroupID:          o.GroupID,
		SpendLimitMicros: o.SpendLimitMicros,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}
}

func InvalidatedPresets(invalidatedPresets []database.UpdatePresetsLastInvalidatedAtRow) []nicloudsdk.InvalidatedPreset {
	var presets []nicloudsdk.InvalidatedPreset
	for _, p := range invalidatedPresets {
		presets = append(presets, nicloudsdk.InvalidatedPreset{
			TemplateName:        p.TemplateName,
			TemplateVersionName: p.TemplateVersionName,
			PresetName:          p.TemplateVersionPresetName,
		})
	}
	return presets
}

// sanitizeCredentialHint ensures the hint looks masked before exposing
// it in the API. The aibridge library uses "..." as the masking
// delimiter (e.g. "sk-a...efgh"), so we check for its presence. If
// the hint doesn't contain "..." or exceeds the max length, it's
// replaced with "..." to prevent leaking raw secrets.
func sanitizeCredentialHint(hint string) string {
	// Matches the VARCHAR(15) DB constraint.
	const maxCredentialHintLength = 15

	if hint == "" {
		return ""
	}

	if len(hint) > maxCredentialHintLength || !strings.Contains(hint, "...") {
		return "..."
	}
	return hint
}

func jsonOrEmptyMap(rawMessage pqtype.NullRawMessage) map[string]any {
	var m map[string]any
	if !rawMessage.Valid {
		return m
	}

	err := json.Unmarshal(rawMessage.RawMessage, &m)
	if err != nil {
		// Don't reuse m
		return map[string]any{}
	}
	return m
}

func ChatMessage(m database.ChatMessage) nicloudsdk.ChatMessage {
	modelConfigID := &m.ModelConfigID.UUID
	if !m.ModelConfigID.Valid {
		modelConfigID = nil
	}
	createdBy := &m.CreatedBy.UUID
	if !m.CreatedBy.Valid {
		createdBy = nil
	}
	msg := nicloudsdk.ChatMessage{
		ID:            m.ID,
		ChatID:        m.ChatID,
		CreatedBy:     createdBy,
		ModelConfigID: modelConfigID,
		CreatedAt:     m.CreatedAt,
		Role:          nicloudsdk.ChatMessageRole(m.Role),
	}
	if m.Content.Valid {
		parts, err := chatMessageParts(m)
		if err == nil {
			msg.Content = parts
		}
	}
	usage := chatMessageUsage(m)
	if usage != nil {
		msg.Usage = usage
	}
	return msg
}

// chatMessageUsage builds a ChatMessageUsage from the database row,
// returning nil when no token fields are populated.
func chatMessageUsage(m database.ChatMessage) *nicloudsdk.ChatMessageUsage {
	inputTokens := nullInt64Ptr(m.InputTokens)
	outputTokens := nullInt64Ptr(m.OutputTokens)
	totalTokens := nullInt64Ptr(m.TotalTokens)
	reasoningTokens := nullInt64Ptr(m.ReasoningTokens)
	cacheCreationTokens := nullInt64Ptr(m.CacheCreationTokens)
	cacheReadTokens := nullInt64Ptr(m.CacheReadTokens)
	contextLimit := nullInt64Ptr(m.ContextLimit)

	if inputTokens == nil && outputTokens == nil && totalTokens == nil &&
		reasoningTokens == nil && cacheCreationTokens == nil &&
		cacheReadTokens == nil && contextLimit == nil {
		return nil
	}

	return &nicloudsdk.ChatMessageUsage{
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		TotalTokens:         totalTokens,
		ReasoningTokens:     reasoningTokens,
		CacheCreationTokens: cacheCreationTokens,
		CacheReadTokens:     cacheReadTokens,
		ContextLimit:        contextLimit,
	}
}

// ChatQueuedMessage converts a queued message to its SDK representation.
func ChatQueuedMessage(message database.ChatQueuedMessage) nicloudsdk.ChatQueuedMessage {
	// Queued messages are always written by current code via
	// MarshalParts, so they are always current content version.
	parts, err := chatMessageParts(database.ChatMessage{
		Role: database.ChatMessageRoleUser,
		Content: pqtype.NullRawMessage{
			RawMessage: message.Content,
			Valid:      len(message.Content) > 0,
		},
		ContentVersion: chatprompt.CurrentContentVersion,
	})
	if err != nil {
		parts = nil
	}

	return nicloudsdk.ChatQueuedMessage{
		ID:            message.ID,
		ChatID:        message.ChatID,
		ModelConfigID: nullUUIDPtr(message.ModelConfigID),
		Content:       parts,
		CreatedAt:     message.CreatedAt,
	}
}

// ChatQueuedMessages converts a slice of database queued messages
// to their SDK representation.
func ChatQueuedMessages(messages []database.ChatQueuedMessage) []nicloudsdk.ChatQueuedMessage {
	out := make([]nicloudsdk.ChatQueuedMessage, 0, len(messages))
	for _, message := range messages {
		out = append(out, ChatQueuedMessage(message))
	}
	return out
}

func chatMessageParts(m database.ChatMessage) ([]nicloudsdk.ChatMessagePart, error) {
	parts, err := chatprompt.ParseContent(m)
	if err != nil {
		return nil, err
	}
	// Strip internal-only fields before API responses.
	for i := range parts {
		parts[i].StripInternal()
	}
	return parts, nil
}

func nullUUIDPtr(v uuid.NullUUID) *uuid.UUID {
	if !v.Valid {
		return nil
	}
	value := v.UUID
	return &value
}

func nullInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	value := v.Int64
	return &value
}

func nullInt32Ptr(n sql.NullInt32) *int32 {
	if !n.Valid {
		return nil
	}
	return &n.Int32
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	value := v.Time
	return &value
}

const fallbackChatLastErrorMessage = "The chat request failed unexpectedly."

func decodeChatLastError(raw pqtype.NullRawMessage) *nicloudsdk.ChatError {
	if !raw.Valid {
		return nil
	}

	var payload nicloudsdk.ChatError
	if err := json.Unmarshal(raw.RawMessage, &payload); err != nil {
		return &nicloudsdk.ChatError{
			Message: fallbackChatLastErrorMessage,
			Kind:    nicloudsdk.ChatErrorKindGeneric,
		}
	}

	payload.Message = strings.TrimSpace(payload.Message)
	payload.Detail = strings.TrimSpace(payload.Detail)
	payload.Kind = nicloudsdk.ChatErrorKind(strings.TrimSpace(string(payload.Kind)))
	payload.Provider = strings.TrimSpace(payload.Provider)
	if payload.Kind == "" {
		payload.Kind = nicloudsdk.ChatErrorKindGeneric
	}
	if payload.Message == "" {
		payload.Message = fallbackChatLastErrorMessage
	}
	return &payload
}

// Chat converts a database.Chat to a nicloudsdk.Chat. It coalesces
// nil slices and maps to empty values for JSON serialization and
// derives RootChatID from the parent chain when not explicitly set.
// When diffStatus is non-nil the response includes diff metadata.
// When files is non-empty the response includes file metadata;
// pass nil to omit the files field (e.g. list endpoints).
func Chat(c database.Chat, diffStatus *database.ChatDiffStatus, files []database.GetChatFileMetadataByChatIDRow) nicloudsdk.Chat {
	mcpServerIDs := c.MCPServerIDs
	if mcpServerIDs == nil {
		mcpServerIDs = []uuid.UUID{}
	}
	labels := map[string]string(c.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	lastError := decodeChatLastError(c.LastError)
	chat := nicloudsdk.Chat{
		ID:                c.ID,
		OrganizationID:    c.OrganizationID,
		OwnerID:           c.OwnerID,
		OwnerUsername:     c.OwnerUsername,
		OwnerName:         c.OwnerName,
		LastModelConfigID: c.LastModelConfigID,
		Title:             c.Title,
		Status:            nicloudsdk.ChatStatus(c.Status),
		Archived:          c.Archived,
		PinOrder:          c.PinOrder,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
		MCPServerIDs:      mcpServerIDs,
		Labels:            labels,
		ClientType:        nicloudsdk.ChatClientType(c.ClientType),
		LastError:         lastError,
	}
	if c.LastTurnSummary.Valid {
		chat.LastTurnSummary = &c.LastTurnSummary.String
	}
	if c.PlanMode.Valid {
		chat.PlanMode = nicloudsdk.ChatPlanMode(c.PlanMode.ChatPlanMode)
	}
	if c.ParentChatID.Valid {
		parentChatID := c.ParentChatID.UUID
		chat.ParentChatID = &parentChatID
	}
	// Always initialize Children to an empty slice so the JSON
	// field serializes as [] rather than null. Root chats may
	// later have children populated; child chats remain empty
	// because nesting depth is capped at 1.
	chat.Children = []nicloudsdk.Chat{}
	switch {
	case c.RootChatID.Valid:
		rootChatID := c.RootChatID.UUID
		chat.RootChatID = &rootChatID
	case c.ParentChatID.Valid:
		rootChatID := c.ParentChatID.UUID
		chat.RootChatID = &rootChatID
	default:
		rootChatID := c.ID
		chat.RootChatID = &rootChatID
	}
	if c.WorkspaceID.Valid {
		chat.WorkspaceID = &c.WorkspaceID.UUID
	}
	if c.BuildID.Valid {
		chat.BuildID = &c.BuildID.UUID
	}
	if c.AgentID.Valid {
		chat.AgentID = &c.AgentID.UUID
	}
	if diffStatus != nil {
		convertedDiffStatus := ChatDiffStatus(c.ID, diffStatus)
		chat.DiffStatus = &convertedDiffStatus
	}
	if len(files) > 0 {
		chat.Files = make([]nicloudsdk.ChatFileMetadata, 0, len(files))
		for _, row := range files {
			chat.Files = append(chat.Files, nicloudsdk.ChatFileMetadata{
				ID:             row.ID,
				OwnerID:        row.OwnerID,
				OrganizationID: row.OrganizationID,
				Name:           row.Name,
				MimeType:       row.Mimetype,
				CreatedAt:      row.CreatedAt,
			})
		}
	}
	if c.LastInjectedContext.Valid {
		var parts []nicloudsdk.ChatMessagePart
		// Internal fields are stripped at write time in
		// chatd.updateLastInjectedContext, so no
		// StripInternal call is needed here. Unmarshal
		// errors are suppressed — the column is written by
		// us with a known schema.
		if err := json.Unmarshal(c.LastInjectedContext.RawMessage, &parts); err == nil {
			chat.LastInjectedContext = parts
		}
	}
	return chat
}

func chatDebugAttempts(raw json.RawMessage) []map[string]any {
	if len(raw) == 0 {
		return nil
	}

	var attempts []map[string]any
	if err := json.Unmarshal(raw, &attempts); err != nil {
		return []map[string]any{{
			"error":       "malformed attempts payload",
			"parse_error": err.Error(),
			"raw":         string(raw),
		}}
	}
	// Guard against JSON literal "null" which unmarshals successfully
	// but leaves the slice nil. The DB column is JSONB NOT NULL but
	// that only rejects SQL NULL, not JSONB null.
	if attempts == nil {
		return []map[string]any{}
	}
	return attempts
}

// rawJSONObject deserializes a JSON object payload for debug display.
// If the payload is malformed, it returns a map with "error" and "raw"
// keys preserving the original content for diagnostics.  Callers that
// consume the result programmatically should check for the "error" key.
func rawJSONObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}

	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return map[string]any{
			"error":       "malformed debug payload",
			"parse_error": err.Error(),
			"raw":         string(raw),
		}
	}
	// Guard against JSON literal "null" which unmarshals successfully
	// but leaves the map nil. The DB column is JSONB NOT NULL but
	// that only rejects SQL NULL, not JSONB null.
	if object == nil {
		return map[string]any{}
	}
	return object
}

func nullRawJSONObject(raw pqtype.NullRawMessage) map[string]any {
	if !raw.Valid {
		return nil
	}
	return rawJSONObject(raw.RawMessage)
}

// ChatDebugRunSummary converts a database.ChatDebugRun to a
// nicloudsdk.ChatDebugRunSummary.
func ChatDebugRunSummary(r database.ChatDebugRun) nicloudsdk.ChatDebugRunSummary {
	return nicloudsdk.ChatDebugRunSummary{
		ID:         r.ID,
		ChatID:     r.ChatID,
		Kind:       nicloudsdk.ChatDebugRunKind(r.Kind),
		Status:     nicloudsdk.ChatDebugStatus(r.Status),
		Provider:   nullStringPtr(r.Provider),
		Model:      nullStringPtr(r.Model),
		Summary:    rawJSONObject(r.Summary),
		StartedAt:  r.StartedAt,
		UpdatedAt:  r.UpdatedAt,
		FinishedAt: nullTimePtr(r.FinishedAt),
	}
}

// ChatDebugStep converts a database.ChatDebugStep to a
// nicloudsdk.ChatDebugStep.
func ChatDebugStep(s database.ChatDebugStep) nicloudsdk.ChatDebugStep {
	return nicloudsdk.ChatDebugStep{
		ID:                  s.ID,
		RunID:               s.RunID,
		ChatID:              s.ChatID,
		StepNumber:          s.StepNumber,
		Operation:           nicloudsdk.ChatDebugStepOperation(s.Operation),
		Status:              nicloudsdk.ChatDebugStatus(s.Status),
		HistoryTipMessageID: nullInt64Ptr(s.HistoryTipMessageID),
		AssistantMessageID:  nullInt64Ptr(s.AssistantMessageID),
		NormalizedRequest:   rawJSONObject(s.NormalizedRequest),
		NormalizedResponse:  nullRawJSONObject(s.NormalizedResponse),
		Usage:               nullRawJSONObject(s.Usage),
		Attempts:            chatDebugAttempts(s.Attempts),
		Error:               nullRawJSONObject(s.Error),
		Metadata:            rawJSONObject(s.Metadata),
		StartedAt:           s.StartedAt,
		UpdatedAt:           s.UpdatedAt,
		FinishedAt:          nullTimePtr(s.FinishedAt),
	}
}

// ChatDebugRunDetail converts a database.ChatDebugRun and its steps
// to a nicloudsdk.ChatDebugRun.
func ChatDebugRunDetail(r database.ChatDebugRun, steps []database.ChatDebugStep) nicloudsdk.ChatDebugRun {
	sdkSteps := make([]nicloudsdk.ChatDebugStep, 0, len(steps))
	for _, s := range steps {
		sdkSteps = append(sdkSteps, ChatDebugStep(s))
	}
	return nicloudsdk.ChatDebugRun{
		ID:                  r.ID,
		ChatID:              r.ChatID,
		RootChatID:          nullUUIDPtr(r.RootChatID),
		ParentChatID:        nullUUIDPtr(r.ParentChatID),
		ModelConfigID:       nullUUIDPtr(r.ModelConfigID),
		TriggerMessageID:    nullInt64Ptr(r.TriggerMessageID),
		HistoryTipMessageID: nullInt64Ptr(r.HistoryTipMessageID),
		Kind:                nicloudsdk.ChatDebugRunKind(r.Kind),
		Status:              nicloudsdk.ChatDebugStatus(r.Status),
		Provider:            nullStringPtr(r.Provider),
		Model:               nullStringPtr(r.Model),
		Summary:             rawJSONObject(r.Summary),
		StartedAt:           r.StartedAt,
		UpdatedAt:           r.UpdatedAt,
		FinishedAt:          nullTimePtr(r.FinishedAt),
		Steps:               sdkSteps,
	}
}

// ChildChatRows converts child chat rows to nicloudsdk.Chat values,
// resolving diff statuses from the shared map. When diffStatuses
// is non-nil, children without an entry receive an empty DiffStatus.
func ChildChatRows(
	children []database.GetChildChatsByParentIDsRow,
	diffStatuses map[uuid.UUID]database.ChatDiffStatus,
) []nicloudsdk.Chat {
	result := make([]nicloudsdk.Chat, len(children))
	for i, row := range children {
		diffStatus, ok := diffStatuses[row.Chat.ID]
		if ok {
			result[i] = Chat(row.Chat, &diffStatus, nil)
		} else {
			result[i] = Chat(row.Chat, nil, nil)
			if diffStatuses != nil {
				emptyDiffStatus := ChatDiffStatus(row.Chat.ID, nil)
				result[i].DiffStatus = &emptyDiffStatus
			}
		}
		result[i].HasUnread = row.HasUnread
	}
	return result
}

// ChatRowsWithChildren converts root chat rows and their child rows
// into nicloudsdk.Chat values with children embedded under each parent.
// Both root and child diff statuses are resolved from the shared map.
func ChatRowsWithChildren(
	roots []database.GetChatsRow,
	children []database.GetChildChatsByParentIDsRow,
	diffStatuses map[uuid.UUID]database.ChatDiffStatus,
) []nicloudsdk.Chat {
	// Group children by parent ID.
	childrenByParent := make(map[uuid.UUID][]database.GetChildChatsByParentIDsRow, len(children))
	for _, row := range children {
		parentID := row.Chat.ParentChatID.UUID
		childrenByParent[parentID] = append(childrenByParent[parentID], row)
	}

	result := make([]nicloudsdk.Chat, len(roots))
	for i, row := range roots {
		diffStatus, ok := diffStatuses[row.Chat.ID]
		if ok {
			result[i] = Chat(row.Chat, &diffStatus, nil)
		} else {
			result[i] = Chat(row.Chat, nil, nil)
			if diffStatuses != nil {
				emptyDiffStatus := ChatDiffStatus(row.Chat.ID, nil)
				result[i].DiffStatus = &emptyDiffStatus
			}
		}
		result[i].HasUnread = row.HasUnread

		// Embed child chats.
		if childRows, ok := childrenByParent[row.Chat.ID]; ok {
			result[i].Children = ChildChatRows(childRows, diffStatuses)
		}
	}
	return result
}

// ChatDiffStatus converts a database.ChatDiffStatus to a
// nicloudsdk.ChatDiffStatus. When status is nil an empty value
// containing only the chatID is returned.
func ChatDiffStatus(chatID uuid.UUID, status *database.ChatDiffStatus) nicloudsdk.ChatDiffStatus {
	result := nicloudsdk.ChatDiffStatus{
		ChatID: chatID,
	}
	if status == nil {
		return result
	}

	result.ChatID = status.ChatID
	if status.Url.Valid {
		u := strings.TrimSpace(status.Url.String)
		if u != "" {
			result.URL = &u
		}
	}
	if result.URL == nil {
		// Try to build a branch URL from the stored origin.
		// Since this function does not have access to the API
		// instance, we construct a GitHub provider directly as
		// a best-effort fallback.
		// TODO: This uses the default github.com API base URL,
		// so branch URLs for GitHub Enterprise instances will
		// be incorrect. To fix this, this function would need
		// access to the external auth configs.
		gp, _ := gitprovider.New("github", "", nil)
		if gp != nil {
			if owner, repo, _, ok := gp.ParseRepositoryOrigin(status.GitRemoteOrigin); ok {
				branchURL := gp.BuildBranchURL(owner, repo, status.GitBranch)
				if branchURL != "" {
					result.URL = &branchURL
				}
			}
		}
	}
	if status.PullRequestState.Valid {
		pullRequestState := strings.TrimSpace(status.PullRequestState.String)
		if pullRequestState != "" {
			result.PullRequestState = &pullRequestState
		}
	}
	result.PullRequestTitle = status.PullRequestTitle
	result.PullRequestDraft = status.PullRequestDraft
	result.ChangesRequested = status.ChangesRequested
	result.Additions = status.Additions
	result.Deletions = status.Deletions
	result.ChangedFiles = status.ChangedFiles
	if status.AuthorLogin.Valid {
		result.AuthorLogin = &status.AuthorLogin.String
	}
	if status.AuthorAvatarUrl.Valid {
		result.AuthorAvatarURL = &status.AuthorAvatarUrl.String
	}
	if status.BaseBranch.Valid {
		result.BaseBranch = &status.BaseBranch.String
	}
	if status.HeadBranch.Valid {
		result.HeadBranch = &status.HeadBranch.String
	}
	if status.PrNumber.Valid {
		result.PRNumber = &status.PrNumber.Int32
	}
	if status.Commits.Valid {
		result.Commits = &status.Commits.Int32
	}
	if status.Approved.Valid {
		result.Approved = &status.Approved.Bool
	}
	if status.ReviewerCount.Valid {
		result.ReviewerCount = &status.ReviewerCount.Int32
	}
	if status.RefreshedAt.Valid {
		refreshedAt := status.RefreshedAt.Time
		result.RefreshedAt = &refreshedAt
	}
	staleAt := status.StaleAt
	result.StaleAt = &staleAt

	return result
}

// UserSecret converts a database ListUserSecretsRow (metadata only,
// no value) to an SDK UserSecret.
func UserSecret(secret database.ListUserSecretsRow) nicloudsdk.UserSecret {
	return nicloudsdk.UserSecret{
		ID:          secret.ID,
		Name:        secret.Name,
		Description: secret.Description,
		EnvName:     secret.EnvName,
		FilePath:    secret.FilePath,
		CreatedAt:   secret.CreatedAt,
		UpdatedAt:   secret.UpdatedAt,
	}
}

// UserSecretFromFull converts a full database UserSecret row to an
// SDK UserSecret, omitting the value and encryption key ID.
func UserSecretFromFull(secret database.UserSecret) nicloudsdk.UserSecret {
	return nicloudsdk.UserSecret{
		ID:          secret.ID,
		Name:        secret.Name,
		Description: secret.Description,
		EnvName:     secret.EnvName,
		FilePath:    secret.FilePath,
		CreatedAt:   secret.CreatedAt,
		UpdatedAt:   secret.UpdatedAt,
	}
}

// UserSecrets converts a slice of database ListUserSecretsRow to
// SDK UserSecret values.
func UserSecrets(secrets []database.ListUserSecretsRow) []nicloudsdk.UserSecret {
	result := make([]nicloudsdk.UserSecret, 0, len(secrets))
	for _, s := range secrets {
		result = append(result, UserSecret(s))
	}
	return result
}

// UserSkill converts a database UserSkill to an SDK UserSkill.
func UserSkill(skill database.UserSkill) nicloudsdk.UserSkill {
	return nicloudsdk.UserSkill{
		UserSkillMetadata: nicloudsdk.UserSkillMetadata{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			CreatedAt:   skill.CreatedAt,
			UpdatedAt:   skill.UpdatedAt,
		},
		Content: skill.Content,
	}
}

// UserSkillMetadata converts database user skill metadata to an SDK UserSkillMetadata.
func UserSkillMetadata(skill database.ListUserSkillMetadataByUserIDRow) nicloudsdk.UserSkillMetadata {
	return nicloudsdk.UserSkillMetadata{
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		CreatedAt:   skill.CreatedAt,
		UpdatedAt:   skill.UpdatedAt,
	}
}

// UserSkillMetadataList converts database user skill metadata rows to SDK values.
func UserSkillMetadataList(rows []database.ListUserSkillMetadataByUserIDRow) []nicloudsdk.UserSkillMetadata {
	metadata := make([]nicloudsdk.UserSkillMetadata, 0, len(rows))
	for _, row := range rows {
		metadata = append(metadata, UserSkillMetadata(row))
	}
	return metadata
}
