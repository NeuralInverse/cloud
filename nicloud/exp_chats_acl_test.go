package nicloud_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbgen"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestChatACLSharingLifecycle(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitLong)
	mAudit := audit.NewMock()
	client, db := newChatClientWithDatabase(t, func(opts *nicloudtest.Options) {
		opts.Auditor = mAudit
	})
	firstUser := nicloudtest.CreateFirstUser(t, client.Client)
	_ = createChatModelConfig(t, client)

	sharedClient, sharedUser := nicloudtest.CreateAnotherUser(t, client.Client, firstUser.OrganizationID)
	sharedClientExp := nicloudsdk.NewExperimentalClient(sharedClient)
	nonSharedClient, _ := nicloudtest.CreateAnotherUser(t, client.Client, firstUser.OrganizationID)
	nonSharedClientExp := nicloudsdk.NewExperimentalClient(nonSharedClient)
	groupMemberClient, groupMember := nicloudtest.CreateAnotherUser(t, client.Client, firstUser.OrganizationID)
	groupMemberClientExp := nicloudsdk.NewExperimentalClient(groupMemberClient)
	sharedGroup := dbgen.Group(t, db, database.Group{OrganizationID: firstUser.OrganizationID})
	dbgen.GroupMember(t, db, database.GroupMemberTable{GroupID: sharedGroup.ID, UserID: groupMember.ID})

	data := []byte("chat sharing file")
	uploaded, err := client.UploadChatFile(ctx, firstUser.OrganizationID, "text/plain", "shared.txt", bytes.NewReader(data))
	require.NoError(t, err)
	chat := createChatForSharing(ctx, t, client, firstUser.OrganizationID, "shared chat", uploaded.ID)

	_, err = sharedClientExp.GetChat(ctx, chat.ID)
	requireSDKError(t, err, http.StatusNotFound)
	_, _, err = nonSharedClientExp.GetChatFile(ctx, uploaded.ID)
	requireSDKError(t, err, http.StatusNotFound)

	err = client.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			sharedUser.ID.String(): nicloudsdk.ChatRoleRead,
		},
		GroupRoles: map[string]nicloudsdk.ChatRole{
			sharedGroup.ID.String(): nicloudsdk.ChatRoleRead,
		},
	})
	require.NoError(t, err)
	require.True(t, mAudit.Contains(t, database.AuditLog{
		Action:       database.AuditActionWrite,
		ResourceType: database.ResourceTypeChat,
		ResourceID:   chat.ID,
		UserID:       firstUser.UserID,
	}))

	acl, err := client.GetChatACL(ctx, chat.ID)
	require.NoError(t, err)
	require.Len(t, acl.Users, 1)
	require.Equal(t, sharedUser.ID.String(), acl.Users[0].ID.String())
	require.Equal(t, map[uuid.UUID]nicloudsdk.ChatRole{
		sharedUser.ID: nicloudsdk.ChatRoleRead,
	}, chatUserRoles(acl.Users))
	require.Equal(t, map[uuid.UUID]nicloudsdk.ChatRole{
		sharedGroup.ID: nicloudsdk.ChatRoleRead,
	}, chatGroupRoles(acl.Groups))
	require.Len(t, acl.Groups, 1)
	require.Equal(t, sharedGroup.ID.String(), acl.Groups[0].ID.String())
	require.Empty(t, acl.Groups[0].Members)
	require.Equal(t, 1, acl.Groups[0].TotalMemberCount)

	sharedACL, err := sharedClientExp.GetChatACL(ctx, chat.ID)
	require.NoError(t, err)
	require.Equal(t, chatUserRoles(acl.Users), chatUserRoles(sharedACL.Users))
	require.Equal(t, chatGroupRoles(acl.Groups), chatGroupRoles(sharedACL.Groups))
	require.Len(t, sharedACL.Groups, 1)
	require.Empty(t, sharedACL.Groups[0].Members)
	require.Equal(t, 1, sharedACL.Groups[0].TotalMemberCount)

	sharedChat, err := sharedClientExp.GetChat(ctx, chat.ID)
	require.NoError(t, err)
	require.Equal(t, chat.ID, sharedChat.ID)
	require.Equal(t, nicloudtest.FirstUserParams.Username, sharedChat.OwnerUsername)
	require.Equal(t, nicloudtest.FirstUserParams.Name, sharedChat.OwnerName)
	require.Len(t, sharedChat.Files, 1)
	require.Equal(t, uploaded.ID, sharedChat.Files[0].ID)

	messages, err := sharedClientExp.GetChatMessages(ctx, chat.ID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, messages.Messages)

	got, contentType, err := sharedClientExp.GetChatFile(ctx, uploaded.ID)
	require.NoError(t, err)
	require.Contains(t, contentType, "text/plain")
	require.Equal(t, data, got)
	_, _, err = nonSharedClientExp.GetChatFile(ctx, uploaded.ID)
	requireSDKError(t, err, http.StatusNotFound)

	groupChat, err := groupMemberClientExp.GetChat(ctx, chat.ID)
	require.NoError(t, err)
	require.Equal(t, chat.ID, groupChat.ID)

	_, err = sharedClientExp.CreateChatMessage(ctx, chat.ID, nicloudsdk.CreateChatMessageRequest{
		Content: []nicloudsdk.ChatInputPart{{
			Type: nicloudsdk.ChatInputPartTypeText,
			Text: "should not send",
		}},
	})
	requireSDKError(t, err, http.StatusNotFound)

	err = sharedClientExp.UpdateChat(ctx, chat.ID, nicloudsdk.UpdateChatRequest{
		Title: ptr.Ref("should not rename"),
	})
	requireSDKError(t, err, http.StatusNotFound)

	err = sharedClientExp.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			groupMember.ID.String(): nicloudsdk.ChatRoleRead,
		},
	})
	requireSDKError(t, err, http.StatusForbidden)

	err = sharedClientExp.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			uuid.NewString(): nicloudsdk.ChatRoleRead,
		},
	})
	requireSDKError(t, err, http.StatusForbidden)

	err = client.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			strings.ToUpper(firstUser.UserID.String()): nicloudsdk.ChatRoleRead,
		},
	})
	sdkErr := requireSDKError(t, err, http.StatusBadRequest)
	require.Equal(t, "Cannot change your own chat sharing role.", sdkErr.Message)

	err = client.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			sharedUser.ID.String(): nicloudsdk.ChatRoleDeleted,
		},
	})
	require.NoError(t, err)
	_, err = sharedClientExp.GetChat(ctx, chat.ID)
	requireSDKError(t, err, http.StatusNotFound)
	_, err = groupMemberClientExp.GetChat(ctx, chat.ID)
	require.NoError(t, err)

	mAudit.ResetLogs()
	err = client.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		GroupRoles: map[string]nicloudsdk.ChatRole{
			sharedGroup.ID.String(): nicloudsdk.ChatRoleDeleted,
		},
	})
	require.NoError(t, err)
	require.True(t, mAudit.Contains(t, database.AuditLog{
		Action:       database.AuditActionWrite,
		ResourceType: database.ResourceTypeChat,
		ResourceID:   chat.ID,
		UserID:       firstUser.UserID,
	}))
	_, err = groupMemberClientExp.GetChat(ctx, chat.ID)
	requireSDKError(t, err, http.StatusNotFound)
}

func TestChatACLSubChatInheritance(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitLong)
	client, db := newChatClientWithDatabase(t)
	firstUser := nicloudtest.CreateFirstUser(t, client.Client)
	modelConfig := createChatModelConfig(t, client)
	sharedClient, sharedUser := nicloudtest.CreateAnotherUser(t, client.Client, firstUser.OrganizationID)
	sharedClientExp := nicloudsdk.NewExperimentalClient(sharedClient)

	root := createChatForSharing(ctx, t, client, firstUser.OrganizationID, "root chat")
	child := dbgen.Chat(t, db, database.Chat{
		OrganizationID:    firstUser.OrganizationID,
		OwnerID:           firstUser.UserID,
		ParentChatID:      uuid.NullUUID{UUID: root.ID, Valid: true},
		LastModelConfigID: modelConfig.ID,
		Title:             "child chat",
	})

	err := client.UpdateChatACL(ctx, root.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			sharedUser.ID.String(): nicloudsdk.ChatRoleRead,
		},
	})
	require.NoError(t, err)

	sharedChild, err := sharedClientExp.GetChat(ctx, child.ID)
	require.NoError(t, err)
	require.Equal(t, child.ID, sharedChild.ID)
	require.NotNil(t, sharedChild.RootChatID)
	require.Equal(t, root.ID, *sharedChild.RootChatID)

	_, err = sharedClientExp.GetChat(ctx, root.ID)
	require.NoError(t, err)

	err = client.UpdateChatACL(ctx, child.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			sharedUser.ID.String(): nicloudsdk.ChatRoleDeleted,
		},
	})
	sdkErr := requireSDKError(t, err, http.StatusBadRequest)
	require.Equal(t, "Chat ACLs can only be set on root chats.", sdkErr.Message)

	_, err = client.GetChatACL(ctx, child.ID)
	sdkErr = requireSDKError(t, err, http.StatusBadRequest)
	require.Equal(t, "Chat ACLs can only be set on root chats.", sdkErr.Message)
}

func TestChatACLValidation(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitLong)
	client := newChatClient(t)
	firstUser := nicloudtest.CreateFirstUser(t, client.Client)
	_ = createChatModelConfig(t, client)
	chat := createChatForSharing(ctx, t, client, firstUser.OrganizationID, "validation chat")
	missingUserID := uuid.New()
	missingGroupID := uuid.New()

	tests := []struct {
		name           string
		req            nicloudsdk.UpdateChatACL
		wantValidation nicloudsdk.ValidationError
	}{
		{
			name: "InvalidRole",
			req: nicloudsdk.UpdateChatACL{
				UserRoles: map[string]nicloudsdk.ChatRole{
					uuid.NewString(): nicloudsdk.ChatRole("write"),
				},
			},
			wantValidation: nicloudsdk.ValidationError{
				Field:  "user_roles",
				Detail: `role "write" is not a valid chat role`,
			},
		},
		{
			name: "InvalidUserUUID",
			req: nicloudsdk.UpdateChatACL{
				UserRoles: map[string]nicloudsdk.ChatRole{
					"not-a-uuid": nicloudsdk.ChatRoleRead,
				},
			},
			wantValidation: nicloudsdk.ValidationError{
				Field:  "user_roles",
				Detail: "not-a-uuid is not a valid UUID.",
			},
		},
		{
			name: "InvalidGroupUUID",
			req: nicloudsdk.UpdateChatACL{
				GroupRoles: map[string]nicloudsdk.ChatRole{
					"not-a-uuid": nicloudsdk.ChatRoleRead,
				},
			},
			wantValidation: nicloudsdk.ValidationError{
				Field:  "group_roles",
				Detail: "not-a-uuid is not a valid UUID.",
			},
		},
		{
			name: "MissingUser",
			req: nicloudsdk.UpdateChatACL{
				UserRoles: map[string]nicloudsdk.ChatRole{
					missingUserID.String(): nicloudsdk.ChatRoleRead,
				},
			},
			wantValidation: nicloudsdk.ValidationError{
				Field:  "user_roles",
				Detail: "user with ID " + missingUserID.String() + " does not exist",
			},
		},
		{
			name: "MissingGroup",
			req: nicloudsdk.UpdateChatACL{
				GroupRoles: map[string]nicloudsdk.ChatRole{
					missingGroupID.String(): nicloudsdk.ChatRoleRead,
				},
			},
			wantValidation: nicloudsdk.ValidationError{
				Field:  "group_roles",
				Detail: "group with ID " + missingGroupID.String() + " does not exist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t, testutil.WaitLong)
			err := client.UpdateChatACL(ctx, chat.ID, tt.req)
			sdkErr := requireSDKError(t, err, http.StatusBadRequest)
			require.Equal(t, "Invalid request to update chat ACL.", sdkErr.Message)
			require.Contains(t, sdkErr.Validations, tt.wantValidation)
		})
	}
}

func TestSharedReaderStreamChat(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitLong)
	client, db := newChatClientWithDatabase(t)
	firstUser := nicloudtest.CreateFirstUser(t, client.Client)
	modelConfig := createChatModelConfig(t, client)
	sharedClient, sharedUser := nicloudtest.CreateAnotherUser(t, client.Client, firstUser.OrganizationID)
	sharedClientExp := nicloudsdk.NewExperimentalClient(sharedClient)
	chat := dbgen.Chat(t, db, database.Chat{
		OrganizationID:    firstUser.OrganizationID,
		OwnerID:           firstUser.UserID,
		LastModelConfigID: modelConfig.ID,
		Title:             "shared stream chat",
	})
	insertAssistantCostMessage(t, db, chat.ID, modelConfig.ID, 0)

	err := client.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			sharedUser.ID.String(): nicloudsdk.ChatRoleRead,
		},
	})
	require.NoError(t, err)

	events, closer, err := sharedClientExp.StreamChat(ctx, chat.ID, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = closer.Close() })

	foundAssistantMessage := false
	for !foundAssistantMessage {
		select {
		case <-ctx.Done():
			require.FailNow(t, "timed out waiting for shared stream chat event")
		case event, ok := <-events:
			require.True(t, ok, "stream closed before expected event")
			require.Equal(t, chat.ID, event.ChatID)
			require.NotEqual(t, nicloudsdk.ChatStreamEventTypeError, event.Type)
			if event.Type == nicloudsdk.ChatStreamEventTypeMessage &&
				event.Message != nil &&
				event.Message.Role == nicloudsdk.ChatMessageRoleAssistant {
				foundAssistantMessage = true
			}
		}
	}
	require.NoError(t, closer.Close())

	persisted, err := db.GetChatByID(dbauthz.AsSystemRestricted(ctx), chat.ID)
	require.NoError(t, err)
	require.False(t, persisted.LastReadMessageID.Valid)
}

func TestListChatsExcludesSharedChats(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitLong)
	client, db := newChatClientWithDatabase(t)
	firstUser := nicloudtest.CreateFirstUser(t, client.Client)
	modelConfig := createChatModelConfig(t, client)
	viewerClient, viewer := nicloudtest.CreateAnotherUser(t, client.Client, firstUser.OrganizationID, rbac.ScopedRoleAgentsAccess(firstUser.OrganizationID))
	viewerClientExp := nicloudsdk.NewExperimentalClient(viewerClient)
	sharedChat := dbgen.Chat(t, db, database.Chat{
		OrganizationID:    firstUser.OrganizationID,
		OwnerID:           firstUser.UserID,
		LastModelConfigID: modelConfig.ID,
		Title:             "shared with viewer",
	})
	viewerChat := dbgen.Chat(t, db, database.Chat{
		OrganizationID:    firstUser.OrganizationID,
		OwnerID:           viewer.ID,
		LastModelConfigID: modelConfig.ID,
		Title:             "viewer owned",
	})

	err := client.UpdateChatACL(ctx, sharedChat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			viewer.ID.String(): nicloudsdk.ChatRoleRead,
		},
	})
	require.NoError(t, err)

	ownedOnly, err := viewerClientExp.ListChats(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, map[uuid.UUID]struct{}{viewerChat.ID: {}}, chatIDSet(ownedOnly))
}

//nolint:paralleltest // This test verifies a process-wide RBAC kill switch.
func TestChatSharingDisabled(t *testing.T) {
	previous := rbac.ChatACLDisabled()
	rbac.SetChatACLDisabled(false)
	rbac.ReloadBuiltinRoles(nil)
	t.Cleanup(func() {
		rbac.ReloadBuiltinRoles(nil)
		rbac.SetChatACLDisabled(previous)
	})

	ctx := testutil.Context(t, testutil.WaitLong)
	values := chatDeploymentValues(t)
	values.DisableChatSharing = true
	store, pubsub := dbtestutil.NewDB(t)
	client := newChatClient(t, func(opts *nicloudtest.Options) {
		opts.DeploymentValues = values
		opts.Database = store
		opts.Pubsub = pubsub
	})
	firstUser := nicloudtest.CreateFirstUser(t, client.Client)
	modelConfig := createChatModelConfig(t, client)
	viewerClient, viewer := nicloudtest.CreateAnotherUser(t, client.Client, firstUser.OrganizationID, rbac.ScopedRoleAgentsAccess(firstUser.OrganizationID))
	viewerClientExp := nicloudsdk.NewExperimentalClient(viewerClient)

	chat := dbgen.Chat(t, store, database.Chat{
		OrganizationID:    firstUser.OrganizationID,
		OwnerID:           firstUser.UserID,
		LastModelConfigID: modelConfig.ID,
		Title:             "disabled sharing",
	})
	err := store.UpdateChatACLByID(ctx, database.UpdateChatACLByIDParams{
		ID: chat.ID,
		UserACL: database.ChatACL{
			viewer.ID.String(): database.ChatACLEntry{Permissions: []policy.Action{policy.ActionRead}},
		},
		GroupACL: database.ChatACL{},
	})
	require.NoError(t, err)

	_, err = viewerClientExp.GetChat(ctx, chat.ID)
	requireSDKError(t, err, http.StatusNotFound)

	_, err = client.GetChatACL(ctx, chat.ID)
	sdkErr := requireSDKError(t, err, http.StatusForbidden)
	require.Equal(t, "Chat sharing is disabled for this deployment.", sdkErr.Message)

	err = client.UpdateChatACL(ctx, chat.ID, nicloudsdk.UpdateChatACL{
		UserRoles: map[string]nicloudsdk.ChatRole{
			viewer.ID.String(): nicloudsdk.ChatRoleRead,
		},
	})
	requireSDKError(t, err, http.StatusForbidden)

	ownerChats, err := client.ListChats(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, map[uuid.UUID]struct{}{chat.ID: {}}, chatIDSet(ownerChats))

	viewerChats, err := viewerClientExp.ListChats(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, viewerChats)
}

func createChatForSharing(
	ctx context.Context,
	t *testing.T,
	client *nicloudsdk.ExperimentalClient,
	organizationID uuid.UUID,
	text string,
	fileIDs ...uuid.UUID,
) nicloudsdk.Chat {
	t.Helper()

	content := []nicloudsdk.ChatInputPart{{
		Type: nicloudsdk.ChatInputPartTypeText,
		Text: text,
	}}
	for _, fileID := range fileIDs {
		content = append(content, nicloudsdk.ChatInputPart{
			Type:   nicloudsdk.ChatInputPartTypeFile,
			FileID: fileID,
		})
	}
	chat, err := client.CreateChat(ctx, nicloudsdk.CreateChatRequest{
		OrganizationID: organizationID,
		Content:        content,
	})
	require.NoError(t, err)
	return chat
}

func chatUserRoles(users []nicloudsdk.ChatUser) map[uuid.UUID]nicloudsdk.ChatRole {
	roles := make(map[uuid.UUID]nicloudsdk.ChatRole, len(users))
	for _, user := range users {
		roles[user.ID] = user.Role
	}
	return roles
}

func chatGroupRoles(groups []nicloudsdk.ChatGroup) map[uuid.UUID]nicloudsdk.ChatRole {
	roles := make(map[uuid.UUID]nicloudsdk.ChatRole, len(groups))
	for _, group := range groups {
		roles[group.ID] = group.Role
	}
	return roles
}

func chatIDSet(chats []nicloudsdk.Chat) map[uuid.UUID]struct{} {
	ids := make(map[uuid.UUID]struct{}, len(chats))
	for _, chat := range chats {
		ids[chat.ID] = struct{}{}
	}
	return ids
}
