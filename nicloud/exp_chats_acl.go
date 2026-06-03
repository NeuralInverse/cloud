package nicloud

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	slog "cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/db2sdk"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/acl"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/slice"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// EXPERIMENTAL: this endpoint is experimental and is subject to change.
//
// @Summary Get chat ACLs
// @ID get-chat-acls
// @Security Neural Inverse CloudSessionToken
// @Tags Chats
// @Produce json
// @Param chat path string true "Chat ID" format(uuid)
// @Success 200 {object} nicloudsdk.ChatACL
// @Router /api/experimental/chats/{chat}/acl [get]
// @x-apidocgen {"skip": true}
// @Description Experimental: this endpoint is subject to change.
//
//nolint:revive // get-return: revive assumes get* must be a getter, but this is an HTTP handler.
func (api *API) getChatACL(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chat := httpmw.ChatParam(r)

	if !api.allowChatSharing(ctx, rw) {
		return
	}
	if chat.IsSubChat() {
		resp := nicloudsdk.Response{Message: "Chat ACLs can only be set on root chats."}
		if chat.RootChatID.Valid {
			resp.Detail = "Target the root chat (id: " + chat.RootChatID.UUID.String() + ") instead."
		}
		httpapi.Write(ctx, rw, http.StatusBadRequest, resp)
		return
	}

	chatACL, err := api.Database.GetChatACLByID(ctx, chat.ID)
	if err != nil {
		if dbauthz.IsNotAuthorizedError(err) {
			httpapi.ResourceNotFound(rw)
			return
		}
		httpapi.InternalServerError(rw, err)
		return
	}

	users, ok := api.chatACLUsers(ctx, rw, chat, chatACL.Users)
	if !ok {
		return
	}
	groups, ok := api.chatACLGroups(ctx, rw, chat, chatACL.Groups)
	if !ok {
		return
	}

	httpapi.Write(ctx, rw, http.StatusOK, nicloudsdk.ChatACL{
		Users:  users,
		Groups: groups,
	})
}

// EXPERIMENTAL: this endpoint is experimental and is subject to change.
//
// @Summary Update chat ACL
// @ID update-chat-acl
// @Security Neural Inverse CloudSessionToken
// @Tags Chats
// @Accept json
// @Param chat path string true "Chat ID" format(uuid)
// @Param request body nicloudsdk.UpdateChatACL true "Update chat ACL request"
// @Success 204
// @Router /api/experimental/chats/{chat}/acl [patch]
// @x-apidocgen {"skip": true}
// @Description Experimental: this endpoint is subject to change.
func (api *API) patchChatACL(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chat := httpmw.ChatParam(r)
	auditor := api.Auditor.Load()
	aReq, commitAudit := audit.InitRequest[database.Chat](rw, &audit.RequestParams{
		Audit:          *auditor,
		Log:            api.Logger,
		Request:        r,
		Action:         database.AuditActionWrite,
		OrganizationID: chat.OrganizationID,
	})
	defer commitAudit()
	aReq.Old = chat

	if !api.allowChatSharing(ctx, rw) {
		return
	}
	if chat.IsSubChat() {
		resp := nicloudsdk.Response{Message: "Chat ACLs can only be set on root chats."}
		if chat.RootChatID.Valid {
			resp.Detail = "Target the root chat (id: " + chat.RootChatID.UUID.String() + ") instead."
		}
		httpapi.Write(ctx, rw, http.StatusBadRequest, resp)
		return
	}
	if !api.Authorize(r, policy.ActionShare, chat.RBACObject()) {
		httpapi.Forbidden(rw)
		return
	}

	var req nicloudsdk.UpdateChatACL
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	apiKey := httpmw.APIKey(r)
	for userID := range req.UserRoles {
		parsed, err := uuid.Parse(userID)
		if err == nil && parsed == apiKey.UserID {
			httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
				Message: "Cannot change your own chat sharing role.",
			})
			return
		}
	}

	validErrs := acl.Validate(ctx, api.Database, ChatACLUpdateValidator(req))
	if len(validErrs) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message:     "Invalid request to update chat ACL.",
			Validations: validErrs,
		})
		return
	}

	err := api.Database.InTx(func(tx database.Store) error {
		current, err := tx.GetChatByIDForUpdate(ctx, chat.ID)
		if err != nil {
			return xerrors.Errorf("get chat by ID: %w", err)
		}
		if current.UserACL == nil {
			current.UserACL = database.ChatACL{}
		}
		if current.GroupACL == nil {
			current.GroupACL = database.ChatACL{}
		}

		for id, role := range req.UserRoles {
			if role == nicloudsdk.ChatRoleDeleted {
				delete(current.UserACL, id)
				continue
			}
			current.UserACL[id] = database.ChatACLEntry{
				Permissions: db2sdk.ChatRoleActions(role),
			}
		}
		for id, role := range req.GroupRoles {
			if role == nicloudsdk.ChatRoleDeleted {
				delete(current.GroupACL, id)
				continue
			}
			current.GroupACL[id] = database.ChatACLEntry{
				Permissions: db2sdk.ChatRoleActions(role),
			}
		}

		if err := tx.UpdateChatACLByID(ctx, database.UpdateChatACLByIDParams{
			ID:       chat.ID,
			UserACL:  current.UserACL,
			GroupACL: current.GroupACL,
		}); err != nil {
			return xerrors.Errorf("update chat ACL: %w", err)
		}
		updatedChat, err := tx.GetChatByID(ctx, chat.ID)
		if err != nil {
			return xerrors.Errorf("get updated chat by ID: %w", err)
		}
		aReq.New = updatedChat
		return nil
	}, nil)
	if err != nil {
		if dbauthz.IsNotAuthorizedError(err) {
			httpapi.Forbidden(rw)
			return
		}
		httpapi.InternalServerError(rw, err)
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

func (api *API) chatACLUsers(ctx context.Context, rw http.ResponseWriter, chat database.Chat, entries database.ChatACL) ([]nicloudsdk.ChatUser, bool) {
	userIDs := make([]uuid.UUID, 0, len(entries))
	for userID := range entries {
		id, err := uuid.Parse(userID)
		if err != nil {
			api.Logger.Warn(ctx, "found invalid user uuid in chat acl", slog.Error(err), slog.F("chat_id", chat.ID))
			continue
		}
		userIDs = append(userIDs, id)
	}

	//nolint:gocritic // Users who can read the chat ACL should see shared users even without user read permission.
	dbUsers, err := api.Database.GetUsersByIDs(dbauthz.AsSystemRestricted(ctx), userIDs)
	if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
		httpapi.InternalServerError(rw, err)
		return nil, false
	}

	users := make([]nicloudsdk.ChatUser, 0, len(dbUsers))
	for _, user := range dbUsers {
		entry := entries[user.ID.String()]
		users = append(users, nicloudsdk.ChatUser{
			MinimalUser: db2sdk.MinimalUser(user),
			Role:        convertToChatRole(entry.Permissions),
		})
	}
	return users, true
}

func (api *API) chatACLGroups(ctx context.Context, rw http.ResponseWriter, chat database.Chat, entries database.ChatACL) ([]nicloudsdk.ChatGroup, bool) {
	groupIDs := make([]uuid.UUID, 0, len(entries))
	for groupID := range entries {
		id, err := uuid.Parse(groupID)
		if err != nil {
			api.Logger.Warn(ctx, "found invalid group uuid in chat acl", slog.Error(err), slog.F("chat_id", chat.ID))
			continue
		}
		groupIDs = append(groupIDs, id)
	}

	dbGroups := make([]database.GetGroupsRow, 0)
	if len(groupIDs) > 0 {
		var err error
		//nolint:gocritic // Users who can read the chat ACL should see shared groups even without group read permission.
		dbGroups, err = api.Database.GetGroups(dbauthz.AsSystemRestricted(ctx), database.GetGroupsParams{GroupIds: groupIDs})
		if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
			httpapi.InternalServerError(rw, err)
			return nil, false
		}
	}

	groups := make([]nicloudsdk.ChatGroup, 0, len(dbGroups))
	for _, group := range dbGroups {
		//nolint:gocritic // Users who can read the chat ACL should see shared group sizes even without group read permission.
		memberCount, err := api.Database.GetGroupMembersCountByGroupID(dbauthz.AsSystemRestricted(ctx), database.GetGroupMembersCountByGroupIDParams{
			GroupID:       group.Group.ID,
			IncludeSystem: false,
		})
		if err != nil {
			httpapi.InternalServerError(rw, err)
			return nil, false
		}
		entry := entries[group.Group.ID.String()]
		groups = append(groups, nicloudsdk.ChatGroup{
			Group: db2sdk.Group(group, nil, int(memberCount)),
			Role:  convertToChatRole(entry.Permissions),
		})
	}
	return groups, true
}

func (api *API) allowChatSharing(ctx context.Context, rw http.ResponseWriter) bool {
	if !api.chatSharingDisabled() {
		return true
	}
	httpapi.Write(ctx, rw, http.StatusForbidden, nicloudsdk.Response{
		Message: "Chat sharing is disabled for this deployment.",
	})
	return false
}

func (api *API) chatSharingDisabled() bool {
	return rbac.ChatACLDisabled() || (api.DeploymentValues != nil && bool(api.DeploymentValues.DisableChatSharing))
}

type ChatACLUpdateValidator nicloudsdk.UpdateChatACL

var _ acl.UpdateValidator[nicloudsdk.ChatRole] = ChatACLUpdateValidator{}

func (c ChatACLUpdateValidator) Users() (map[string]nicloudsdk.ChatRole, string) {
	return c.UserRoles, "user_roles"
}

func (c ChatACLUpdateValidator) Groups() (map[string]nicloudsdk.ChatRole, string) {
	return c.GroupRoles, "group_roles"
}

func (ChatACLUpdateValidator) ValidateRole(role nicloudsdk.ChatRole) error {
	if role == nicloudsdk.ChatRoleDeleted || role == nicloudsdk.ChatRoleRead {
		return nil
	}
	return xerrors.Errorf("role %q is not a valid chat role", role)
}

func convertToChatRole(actions []policy.Action) nicloudsdk.ChatRole {
	if slice.SameElements(actions, db2sdk.ChatRoleActions(nicloudsdk.ChatRoleRead)) {
		return nicloudsdk.ChatRoleRead
	}

	return nicloudsdk.ChatRoleDeleted
}
