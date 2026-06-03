package nicloudenttest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/pubsub"
	"github.com/NeuralInverse/cloud/v2/nicloud/prebuilds"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/namesgenerator"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/drpcsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	entprebuilds "github.com/NeuralInverse/cloud/v2/enterprise/nicloud/prebuilds"
	"github.com/NeuralInverse/cloud/v2/enterprise/dbcrypt"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/provisioner/terraform"
	"github.com/NeuralInverse/cloud/v2/provisionerd"
	provisionerdproto "github.com/NeuralInverse/cloud/v2/provisionerd/proto"
	"github.com/NeuralInverse/cloud/v2/provisionersdk"
	sdkproto "github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

const (
	testKeyID = "enterprise-test"
)

var (
	testPrivateKey ed25519.PrivateKey
	testPublicKey  ed25519.PublicKey

	Keys = map[string]ed25519.PublicKey{}
)

func init() {
	var err error
	testPublicKey, testPrivateKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	Keys[testKeyID] = testPublicKey
}

type Options struct {
	*nicloudtest.Options
	ConnectionLogging          bool
	AuditLogging               bool
	BrowserOnly                bool
	EntitlementsUpdateInterval time.Duration
	SCIMAPIKey                 []byte
	UseLegacySCIM              bool
	UserWorkspaceQuota         int
	ProxyHealthInterval        time.Duration
	LicenseOptions             *LicenseOptions
	DontAddLicense             bool
	DontAddFirstUser           bool
	ReplicaSyncUpdateInterval  time.Duration
	ReplicaErrorGracePeriod    time.Duration
	ExternalTokenEncryption    []dbcrypt.Cipher
	ProvisionerDaemonPSK       string
}

// New constructs a nicloudsdk client connected to an in-memory Enterprise API instance.
func New(t *testing.T, options *Options) (*nicloudsdk.Client, nicloudsdk.CreateFirstUserResponse) {
	client, _, _, user := NewWithAPI(t, options)
	return client, user
}

func NewWithDatabase(t *testing.T, options *Options) (*nicloudsdk.Client, database.Store, nicloudsdk.CreateFirstUserResponse) {
	client, _, api, user := NewWithAPI(t, options)
	return client, api.Database, user
}

func NewWithAPI(t *testing.T, options *Options) (
	*nicloudsdk.Client, io.Closer, *nicloud.API, nicloudsdk.CreateFirstUserResponse,
) {
	t.Helper()

	if options == nil {
		options = &Options{}
	}
	if options.Options == nil {
		options.Options = &nicloudtest.Options{}
	}
	require.False(t, options.DontAddFirstUser && !options.DontAddLicense, "DontAddFirstUser requires DontAddLicense")
	setHandler, cancelFunc, serverURL, oop := nicloudtest.NewOptions(t, options.Options)
	niAPI, err := nicloud.New(context.Background(), &nicloud.Options{
		RBAC:                       true,
		ConnectionLogging:          options.ConnectionLogging,
		AuditLogging:               options.AuditLogging,
		BrowserOnly:                options.BrowserOnly,
		SCIMAPIKey:                 options.SCIMAPIKey,
		UseLegacySCIM:              options.UseLegacySCIM,
		DERPServerRelayAddress:     serverURL.String(),
		DERPServerRegionID:         int(oop.DeploymentValues.DERP.Server.RegionID.Value()),
		ReplicaSyncUpdateInterval:  options.ReplicaSyncUpdateInterval,
		ReplicaErrorGracePeriod:    options.ReplicaErrorGracePeriod,
		Options:                    oop,
		EntitlementsUpdateInterval: options.EntitlementsUpdateInterval,
		LicenseKeys:                Keys,
		ProxyHealthInterval:        options.ProxyHealthInterval,
		DefaultQuietHoursSchedule:  oop.DeploymentValues.UserQuietHoursSchedule.DefaultSchedule.Value(),
		ProvisionerDaemonPSK:       options.ProvisionerDaemonPSK,
		ExternalTokenEncryption:    options.ExternalTokenEncryption,
	})
	require.NoError(t, err)
	setHandler(niAPI.AGPL.RootHandler)
	var provisionerCloser io.Closer = nopcloser{}
	if options.IncludeProvisionerDaemon {
		provisionerCloser = nicloudtest.NewProvisionerDaemon(t, niAPI.AGPL)
	}

	t.Cleanup(func() {
		cancelFunc()
		_ = provisionerCloser.Close()
		_ = niAPI.Close()
	})
	client := nicloudsdk.New(serverURL)
	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		},
	}
	var user nicloudsdk.CreateFirstUserResponse
	if !options.DontAddFirstUser {
		user = nicloudtest.CreateFirstUser(t, client)
		if !options.DontAddLicense {
			lo := LicenseOptions{}
			if options.LicenseOptions != nil {
				lo = *options.LicenseOptions
				// The pgCoord is not supported by the fake DB & in-memory Pubsub.  It only works on a real postgres.
				if lo.AllFeatures || (lo.Features != nil && lo.Features[nicloudsdk.FeatureHighAvailability] != 0) {
					// we check for the in-memory test types so that the real types don't have to exported
					_, ok := niAPI.Pubsub.(*pubsub.MemoryPubsub)
					require.False(t, ok, "FeatureHighAvailability is incompatible with MemoryPubsub")
				}
			}
			_ = AddLicense(t, client, lo)
		}
	}
	return client, provisionerCloser, niAPI, user
}

// LicenseOptions is used to generate a license for testing.
// It supports the builder pattern for easy customization.
type LicenseOptions struct {
	AccountType      string
	AccountID        string
	DeploymentIDs    []string
	Trial            bool
	FeatureSet       nicloudsdk.FeatureSet
	AllFeatures      bool
	PublishUsageData bool
	// GraceAt is the time at which the license will enter the grace period.
	GraceAt time.Time
	// ExpiresAt is the time at which the license will hard expire.
	// ExpiresAt should always be greater then GraceAt.
	ExpiresAt time.Time
	// NotBefore is the time at which the license becomes valid. If set to the
	// zero value, the `nbf` claim on the license is set to 1 minute in the
	// past.
	NotBefore time.Time
	// IssuedAt is the time at which the license was issued. If set to the
	// zero value, the `iat` claim on the license is set to 1 minute in the
	// past.
	IssuedAt time.Time
	Features license.Features
	Addons   []nicloudsdk.Addon

	AllowEmpty bool
}

func (opts *LicenseOptions) WithIssuedAt(now time.Time) *LicenseOptions {
	opts.IssuedAt = now
	return opts
}

func (opts *LicenseOptions) Expired(now time.Time) *LicenseOptions {
	opts.NotBefore = now.Add(time.Hour * 24 * -4) // needs to be before the grace period
	opts.ExpiresAt = now.Add(time.Hour * 24 * -2)
	opts.GraceAt = now.Add(time.Hour * 24 * -3)
	return opts
}

func (opts *LicenseOptions) GracePeriod(now time.Time) *LicenseOptions {
	opts.NotBefore = now.Add(time.Hour * 24 * -2) // needs to be before the grace period
	opts.ExpiresAt = now.Add(time.Hour * 24)
	opts.GraceAt = now.Add(time.Hour * 24 * -1)
	return opts
}

func (opts *LicenseOptions) Valid(now time.Time) *LicenseOptions {
	opts.ExpiresAt = now.Add(time.Hour * 24 * 60)
	opts.GraceAt = now.Add(time.Hour * 24 * 53)
	return opts
}

func (opts *LicenseOptions) FutureTerm(now time.Time) *LicenseOptions {
	opts.NotBefore = now.Add(time.Hour * 24)
	opts.ExpiresAt = now.Add(time.Hour * 24 * 60)
	opts.GraceAt = now.Add(time.Hour * 24 * 53)
	return opts
}

func (opts *LicenseOptions) UserLimit(limit int64) *LicenseOptions {
	return opts.Feature(nicloudsdk.FeatureUserLimit, limit)
}

func (opts *LicenseOptions) AIGovernanceAddon(limit int64) *LicenseOptions {
	opts.Addons = append(opts.Addons, nicloudsdk.AddonAIGovernance)
	return opts.Feature(nicloudsdk.FeatureAIGovernanceUserLimit, limit)
}

func (opts *LicenseOptions) ManagedAgentLimit(limit int64) *LicenseOptions {
	return opts.Feature(nicloudsdk.FeatureManagedAgentLimit, limit)
}

func (opts *LicenseOptions) Feature(name nicloudsdk.FeatureName, value int64) *LicenseOptions {
	if opts.Features == nil {
		opts.Features = license.Features{}
	}
	opts.Features[name] = value
	return opts
}

func (opts *LicenseOptions) Generate(t *testing.T) string {
	return GenerateLicense(t, *opts)
}

// AddFullLicense generates a license with all features enabled.
func AddFullLicense(t *testing.T, client *nicloudsdk.Client) nicloudsdk.License {
	return AddLicense(t, client, LicenseOptions{AllFeatures: true})
}

// AddLicense generates a new license with the options provided and inserts it.
func AddLicense(t *testing.T, client *nicloudsdk.Client, options LicenseOptions) nicloudsdk.License {
	l, err := client.AddLicense(context.Background(), nicloudsdk.AddLicenseRequest{
		License: GenerateLicense(t, options),
	})
	require.NoError(t, err)
	return l
}

// GenerateLicense returns a signed JWT using the test key.
func GenerateLicense(t *testing.T, options LicenseOptions) string {
	t.Helper()
	if options.ExpiresAt.IsZero() {
		options.ExpiresAt = time.Now().Add(time.Hour)
	}
	if options.GraceAt.IsZero() {
		options.GraceAt = time.Now().Add(time.Hour)
	}
	if options.NotBefore.IsZero() {
		options.NotBefore = time.Now().Add(-time.Minute)
	}

	issuedAt := options.IssuedAt
	if issuedAt.IsZero() {
		issuedAt = time.Now().Add(-time.Minute)
	}

	if !options.AllowEmpty && options.AccountType == "" {
		options.AccountType = license.AccountTypeSalesforce
	}
	if !options.AllowEmpty && options.AccountID == "" {
		options.AccountID = "test-account-id"
	}

	c := &license.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "test@testing.test",
			ExpiresAt: jwt.NewNumericDate(options.ExpiresAt),
			NotBefore: jwt.NewNumericDate(options.NotBefore),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
		},
		LicenseExpires:   jwt.NewNumericDate(options.GraceAt),
		AccountType:      options.AccountType,
		AccountID:        options.AccountID,
		DeploymentIDs:    options.DeploymentIDs,
		Trial:            options.Trial,
		Version:          license.CurrentVersion,
		AllFeatures:      options.AllFeatures,
		FeatureSet:       options.FeatureSet,
		Features:         options.Features,
		Addons:           options.Addons,
		PublishUsageData: options.PublishUsageData,
	}
	return GenerateLicenseRaw(t, c)
}

func GenerateLicenseRaw(t *testing.T, claims jwt.Claims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tok.Header[license.HeaderKeyID] = testKeyID
	signedTok, err := tok.SignedString(testPrivateKey)
	require.NoError(t, err)
	return signedTok
}

type nopcloser struct{}

func (nopcloser) Close() error { return nil }

type CreateOrganizationOptions struct {
	// IncludeProvisionerDaemon will spin up an external provisioner for the organization.
	// This requires enterprise and the feature 'nicloudsdk.FeatureExternalProvisionerDaemons'
	IncludeProvisionerDaemon bool
}

func CreateOrganization(t *testing.T, client *nicloudsdk.Client, opts CreateOrganizationOptions, mutators ...func(*nicloudsdk.CreateOrganizationRequest)) nicloudsdk.Organization {
	ctx := testutil.Context(t, testutil.WaitMedium)
	req := nicloudsdk.CreateOrganizationRequest{
		Name:        strings.ToLower(namesgenerator.UniqueNameWith("-")),
		DisplayName: namesgenerator.UniqueName(),
		Description: namesgenerator.UniqueName(),
		Icon:        "",
	}
	for _, mutator := range mutators {
		mutator(&req)
	}

	org, err := client.CreateOrganization(ctx, req)
	require.NoError(t, err)

	if opts.IncludeProvisionerDaemon {
		closer := NewExternalProvisionerDaemon(t, client, org.ID, map[string]string{})
		t.Cleanup(func() {
			_ = closer.Close()
		})
	}

	return org
}

// NewExternalProvisionerDaemon runs an external provisioner daemon in a
// goroutine and returns a closer to stop it. The echo provisioner is used
// here. This is the default provisioner for tests and should be fine for
// most use cases. If you need to test terraform-specific behaviors, use
// NewExternalProvisionerDaemonTerraform instead.
func NewExternalProvisionerDaemon(t testing.TB, client *nicloudsdk.Client, org uuid.UUID, tags map[string]string) io.Closer {
	t.Helper()
	return newExternalProvisionerDaemon(t, client, org, tags, nicloudsdk.ProvisionerTypeEcho)
}

// NewExternalProvisionerDaemonTerraform runs an external provisioner daemon in
// a goroutine and returns a closer to stop it. The terraform provisioner is
// used here. Avoid using this unless you need to test terraform-specific
// behaviors!
func NewExternalProvisionerDaemonTerraform(t testing.TB, client *nicloudsdk.Client, org uuid.UUID, tags map[string]string) io.Closer {
	t.Helper()
	return newExternalProvisionerDaemon(t, client, org, tags, nicloudsdk.ProvisionerTypeTerraform)
}

// nolint // This function is a helper for tests and should not be linted.
func newExternalProvisionerDaemon(t testing.TB, client *nicloudsdk.Client, org uuid.UUID, tags map[string]string, provisionerType nicloudsdk.ProvisionerType) io.Closer {
	t.Helper()

	entitlements, err := client.Entitlements(context.Background())
	if err != nil {
		t.Errorf("external provisioners requires a license with entitlements. The client failed to fetch the entitlements, is this an enterprise instance of nicloud?")
		t.FailNow()
		return nil
	}

	feature := entitlements.Features[nicloudsdk.FeatureExternalProvisionerDaemons]
	if !feature.Enabled || feature.Entitlement != nicloudsdk.EntitlementEntitled {
		t.Errorf("external provisioner daemons require an entitled license")
		t.FailNow()
		return nil
	}

	provisionerClient, provisionerSrv := drpcsdk.MemTransportPipe()
	ctx, cancelFunc := context.WithCancel(context.Background())
	serveDone := make(chan struct{})
	t.Cleanup(func() {
		_ = provisionerClient.Close()
		_ = provisionerSrv.Close()
		cancelFunc()
		<-serveDone
	})

	switch provisionerType {
	case nicloudsdk.ProvisionerTypeTerraform:
		// Ensure the Terraform binary is present in the path.
		// If not, we fail this test rather than downloading it.
		terraformPath, err := exec.LookPath("terraform")
		require.NoError(t, err, "terraform binary not found in PATH")
		t.Logf("using Terraform binary at %s", terraformPath)

		go func() {
			defer close(serveDone)
			assert.NoError(t, terraform.Serve(ctx, &terraform.ServeOptions{
				BinaryPath: terraformPath,
				CachePath:  t.TempDir(),
				ServeOptions: &provisionersdk.ServeOptions{
					Listener:      provisionerSrv,
					WorkDirectory: t.TempDir(),
					Experiments:   nicloudsdk.Experiments{},
				},
			}))
		}()
	case nicloudsdk.ProvisionerTypeEcho:
		go func() {
			defer close(serveDone)
			assert.NoError(t, echo.Serve(ctx, &provisionersdk.ServeOptions{
				Listener:      provisionerSrv,
				WorkDirectory: t.TempDir(),
			}))
		}()
	default:
		t.Fatalf("unsupported provisioner type: %s", provisionerType)
		return nil
	}

	daemon := provisionerd.New(func(ctx context.Context) (provisionerdproto.DRPCProvisionerDaemonClient, error) {
		return client.ServeProvisionerDaemon(ctx, nicloudsdk.ServeProvisionerDaemonRequest{
			Name:         testutil.GetRandomName(t),
			Organization: org,
			Provisioners: []nicloudsdk.ProvisionerType{provisionerType},
			Tags:         tags,
		})
	}, &provisionerd.Options{
		Logger:              testutil.Logger(t).Named("provisionerd").Leveled(slog.LevelDebug),
		UpdateInterval:      250 * time.Millisecond,
		ForceCancelInterval: 5 * time.Second,
		Connector: provisionerd.LocalProvisioners{
			string(provisionerType): sdkproto.NewDRPCProvisionerClient(provisionerClient),
		},
	})
	closer := nicloudtest.NewProvisionerDaemonCloser(daemon)
	t.Cleanup(func() {
		_ = closer.Close()
	})

	return closer
}

func GetRunningPrebuilds(
	ctx context.Context,
	t *testing.T,
	db database.Store,
	desiredPrebuilds int,
) []database.GetRunningPrebuiltWorkspacesRow {
	t.Helper()

	var runningPrebuilds []database.GetRunningPrebuiltWorkspacesRow
	testutil.Eventually(ctx, t, func(context.Context) bool {
		prebuiltWorkspaces, err := db.GetRunningPrebuiltWorkspaces(ctx)
		assert.NoError(t, err, "failed to get running prebuilds")

		for _, prebuild := range prebuiltWorkspaces {
			runningPrebuilds = append(runningPrebuilds, prebuild)

			agents, err := db.GetWorkspaceAgentsInLatestBuildByWorkspaceID(ctx, prebuild.ID)
			assert.NoError(t, err, "failed to get agents")

			// Manually mark all agents as ready since tests don't have real agent processes
			// that would normally report their lifecycle state. Prebuilt workspaces are only
			// eligible for claiming when their agents reach the "ready" state.
			for _, agent := range agents {
				err = db.UpdateWorkspaceAgentLifecycleStateByID(ctx, database.UpdateWorkspaceAgentLifecycleStateByIDParams{
					ID:             agent.ID,
					LifecycleState: database.WorkspaceAgentLifecycleStateReady,
					StartedAt:      sql.NullTime{Time: time.Now().Add(time.Hour), Valid: true},
					ReadyAt:        sql.NullTime{Time: time.Now().Add(-1 * time.Hour), Valid: true},
				})
				assert.NoError(t, err, "failed to update agent")
			}
		}

		t.Logf("found %d running prebuilds so far, want %d", len(runningPrebuilds), desiredPrebuilds)
		return len(runningPrebuilds) == desiredPrebuilds
	}, testutil.IntervalSlow, "found %d running prebuilds, expected %d", len(runningPrebuilds), desiredPrebuilds)

	return runningPrebuilds
}

func MustRunReconciliationLoopForPreset(
	ctx context.Context,
	t *testing.T,
	db database.Store,
	reconciler *entprebuilds.StoreReconciler,
	preset nicloudsdk.Preset,
) []*prebuilds.ReconciliationActions {
	t.Helper()

	state, err := reconciler.SnapshotState(ctx, db)
	require.NoError(t, err)
	ps, err := state.FilterByPreset(preset.ID)
	require.NoError(t, err)
	require.NotNil(t, ps)
	actions, err := reconciler.CalculateActions(ctx, *ps)
	require.NoError(t, err)
	require.NotNil(t, actions)
	require.NoError(t, reconciler.ReconcilePreset(ctx, *ps))

	return actions
}

func MustClaimPrebuild(
	ctx context.Context,
	t *testing.T,
	client *nicloudsdk.Client,
	userClient *nicloudsdk.Client,
	username string,
	version nicloudsdk.TemplateVersion,
	presetID uuid.UUID,
	autostartSchedule ...string,
) nicloudsdk.Workspace {
	t.Helper()

	var startSchedule string
	if len(autostartSchedule) > 0 {
		startSchedule = autostartSchedule[0]
	}

	workspaceName := strings.ReplaceAll(testutil.GetRandomName(t), "_", "-")
	userWorkspace, err := userClient.CreateUserWorkspace(ctx, username, nicloudsdk.CreateWorkspaceRequest{
		TemplateVersionID:       version.ID,
		Name:                    workspaceName,
		TemplateVersionPresetID: presetID,
		AutostartSchedule:       ptr.Ref(startSchedule),
	})
	require.NoError(t, err)
	build := nicloudtest.AwaitWorkspaceBuildJobCompleted(t, userClient, userWorkspace.LatestBuild.ID)
	require.Equal(t, build.Job.Status, nicloudsdk.ProvisionerJobSucceeded)
	workspace := nicloudtest.MustWorkspace(t, client, userWorkspace.ID)
	require.Equal(t, nicloudsdk.WorkspaceTransitionStart, workspace.LatestBuild.Transition)

	return workspace
}
