package cli_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestSharingShare(t *testing.T) {
	t.Parallel()

	t.Run("ShareWithUsers_Simple", func(t *testing.T) {
		t.Parallel()

		var (
			client, db                           = nicloudtest.NewWithDatabase(t, nil)
			orgOwner                             = nicloudtest.CreateFirstUser(t, client)
			workspaceOwnerClient, workspaceOwner = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID, rbac.ScopedRoleOrgAuditor(orgOwner.OrganizationID))
			workspace                            = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				OwnerID:        workspaceOwner.ID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace
			_, toShareWithUser = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
		)

		ctx := testutil.Context(t, testutil.WaitMedium)
		inv, root := clitest.New(t, "sharing", "add", workspace.Name, "--user", toShareWithUser.Username)
		clitest.SetupConfig(t, workspaceOwnerClient, root)

		out := new(bytes.Buffer)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		acl, err := workspaceOwnerClient.WorkspaceACL(inv.Context(), workspace.ID)
		require.NoError(t, err)
		assert.Contains(t, acl.Users, nicloudsdk.WorkspaceUser{
			MinimalUser: nicloudsdk.MinimalUser{
				ID:        toShareWithUser.ID,
				Username:  toShareWithUser.Username,
				Name:      toShareWithUser.Name,
				AvatarURL: toShareWithUser.AvatarURL,
			},
			Role: nicloudsdk.WorkspaceRole("use"),
		})

		assert.Contains(t, out.String(), toShareWithUser.Username)
		assert.Contains(t, out.String(), nicloudsdk.WorkspaceRoleUse)
	})

	t.Run("ShareWithUsers_Multiple", func(t *testing.T) {
		t.Parallel()

		var (
			client, db = nicloudtest.NewWithDatabase(t, nil)
			orgOwner   = nicloudtest.CreateFirstUser(t, client)

			workspaceOwnerClient, workspaceOwner = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID, rbac.ScopedRoleOrgAuditor(orgOwner.OrganizationID))
			workspace                            = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				OwnerID:        workspaceOwner.ID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace

			_, toShareWithUser1 = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
			_, toShareWithUser2 = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
		)

		ctx := testutil.Context(t, testutil.WaitMedium)
		inv, root := clitest.New(t,
			"sharing",
			"add", workspace.Name,
			fmt.Sprintf("--user=%s,%s", toShareWithUser1.Username, toShareWithUser2.Username),
		)
		clitest.SetupConfig(t, workspaceOwnerClient, root)

		out := new(bytes.Buffer)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		acl, err := workspaceOwnerClient.WorkspaceACL(inv.Context(), workspace.ID)
		require.NoError(t, err)
		assert.Contains(t, acl.Users, nicloudsdk.WorkspaceUser{
			MinimalUser: nicloudsdk.MinimalUser{
				ID:        toShareWithUser1.ID,
				Username:  toShareWithUser1.Username,
				Name:      toShareWithUser1.Name,
				AvatarURL: toShareWithUser1.AvatarURL,
			},
			Role: nicloudsdk.WorkspaceRoleUse,
		})
		assert.Contains(t, acl.Users, nicloudsdk.WorkspaceUser{
			MinimalUser: nicloudsdk.MinimalUser{
				ID:        toShareWithUser2.ID,
				Username:  toShareWithUser2.Username,
				Name:      toShareWithUser2.Name,
				AvatarURL: toShareWithUser2.AvatarURL,
			},
			Role: nicloudsdk.WorkspaceRoleUse,
		})

		assert.Contains(t, out.String(), toShareWithUser1.Username)
		assert.Contains(t, out.String(), toShareWithUser2.Username)
	})

	t.Run("ShareWithUsers_Roles", func(t *testing.T) {
		t.Parallel()

		var (
			client, db                           = nicloudtest.NewWithDatabase(t, nil)
			orgOwner                             = nicloudtest.CreateFirstUser(t, client)
			workspaceOwnerClient, workspaceOwner = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID, rbac.ScopedRoleOrgAuditor(orgOwner.OrganizationID))
			workspace                            = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				OwnerID:        workspaceOwner.ID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace
			_, toShareWithUser = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
		)

		ctx := testutil.Context(t, testutil.WaitMedium)
		inv, root := clitest.New(t, "sharing", "add", workspace.Name,
			"--user", fmt.Sprintf("%s:admin", toShareWithUser.Username),
		)
		clitest.SetupConfig(t, workspaceOwnerClient, root)

		out := new(bytes.Buffer)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		acl, err := workspaceOwnerClient.WorkspaceACL(inv.Context(), workspace.ID)
		require.NoError(t, err)
		assert.Contains(t, acl.Users, nicloudsdk.WorkspaceUser{
			MinimalUser: nicloudsdk.MinimalUser{
				ID:        toShareWithUser.ID,
				Username:  toShareWithUser.Username,
				Name:      toShareWithUser.Name,
				AvatarURL: toShareWithUser.AvatarURL,
			},
			Role: nicloudsdk.WorkspaceRoleAdmin,
		})

		found := false
		for _, line := range strings.Split(out.String(), "\n") {
			if strings.Contains(line, toShareWithUser.Username) && strings.Contains(line, string(nicloudsdk.WorkspaceRoleAdmin)) {
				found = true
				break
			}
		}
		assert.True(t, found, fmt.Sprintf("expected to find the username %s and role %s in the command: %s", toShareWithUser.Username, nicloudsdk.WorkspaceRoleAdmin, out.String()))
	})
}

func TestSharingStatus(t *testing.T) {
	t.Parallel()

	t.Run("ListSharedUsers", func(t *testing.T) {
		t.Parallel()

		var (
			client, db                           = nicloudtest.NewWithDatabase(t, nil)
			orgOwner                             = nicloudtest.CreateFirstUser(t, client)
			workspaceOwnerClient, workspaceOwner = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID, rbac.ScopedRoleOrgAuditor(orgOwner.OrganizationID))
			workspace                            = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				OwnerID:        workspaceOwner.ID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace
			_, toShareWithUser = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
			ctx                = testutil.Context(t, testutil.WaitMedium)
		)

		err := client.UpdateWorkspaceACL(ctx, workspace.ID, nicloudsdk.UpdateWorkspaceACL{
			UserRoles: map[string]nicloudsdk.WorkspaceRole{
				toShareWithUser.ID.String(): nicloudsdk.WorkspaceRoleUse,
			},
		})
		require.NoError(t, err)

		inv, root := clitest.New(t, "sharing", "status", workspace.Name)
		clitest.SetupConfig(t, workspaceOwnerClient, root)

		out := new(bytes.Buffer)
		inv.Stdout = out
		err = inv.WithContext(ctx).Run()
		require.NoError(t, err)

		found := false
		for _, line := range strings.Split(out.String(), "\n") {
			if strings.Contains(line, toShareWithUser.Username) && strings.Contains(line, string(nicloudsdk.WorkspaceRoleUse)) {
				found = true
				break
			}
		}
		assert.True(t, found, "expected to find username %s with role %s in the output: %s", toShareWithUser.Username, nicloudsdk.WorkspaceRoleUse, out.String())
	})
}

func TestSharingRemove(t *testing.T) {
	t.Parallel()

	t.Run("RemoveSharedUser_Simple", func(t *testing.T) {
		t.Parallel()

		var (
			client, db                           = nicloudtest.NewWithDatabase(t, nil)
			orgOwner                             = nicloudtest.CreateFirstUser(t, client)
			workspaceOwnerClient, workspaceOwner = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID, rbac.ScopedRoleOrgAuditor(orgOwner.OrganizationID))
			workspace                            = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				OwnerID:        workspaceOwner.ID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace
			_, toRemoveUser    = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
			_, toShareWithUser = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
		)

		ctx := testutil.Context(t, testutil.WaitMedium)

		// Share the workspace with a user to later remove
		err := client.UpdateWorkspaceACL(ctx, workspace.ID, nicloudsdk.UpdateWorkspaceACL{
			UserRoles: map[string]nicloudsdk.WorkspaceRole{
				toShareWithUser.ID.String(): nicloudsdk.WorkspaceRoleUse,
				toRemoveUser.ID.String():    nicloudsdk.WorkspaceRoleUse,
			},
		})
		require.NoError(t, err)

		inv, root := clitest.New(t,
			"sharing",
			"remove",
			workspace.Name,
			"--user", toRemoveUser.Username,
		)
		clitest.SetupConfig(t, workspaceOwnerClient, root)

		out := new(bytes.Buffer)
		inv.Stdout = out
		err = inv.WithContext(ctx).Run()
		require.NoError(t, err)

		acl, err := workspaceOwnerClient.WorkspaceACL(inv.Context(), workspace.ID)
		require.NoError(t, err)

		removedCorrectUser := true
		keptOtherUser := false
		for _, user := range acl.Users {
			if user.ID == toRemoveUser.ID {
				removedCorrectUser = false
			}

			if user.ID == toShareWithUser.ID {
				keptOtherUser = true
			}
		}
		assert.True(t, removedCorrectUser)
		assert.True(t, keptOtherUser)
	})

	t.Run("RemoveSharedUser_Multiple", func(t *testing.T) {
		t.Parallel()

		var (
			client, db                           = nicloudtest.NewWithDatabase(t, nil)
			orgOwner                             = nicloudtest.CreateFirstUser(t, client)
			workspaceOwnerClient, workspaceOwner = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID, rbac.ScopedRoleOrgAuditor(orgOwner.OrganizationID))
			workspace                            = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				OwnerID:        workspaceOwner.ID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace
			_, toRemoveUser1 = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
			_, toRemoveUser2 = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID)
		)

		ctx := testutil.Context(t, testutil.WaitMedium)

		// Share the workspace with a user to later remove
		err := client.UpdateWorkspaceACL(ctx, workspace.ID, nicloudsdk.UpdateWorkspaceACL{
			UserRoles: map[string]nicloudsdk.WorkspaceRole{
				toRemoveUser2.ID.String(): nicloudsdk.WorkspaceRoleUse,
				toRemoveUser1.ID.String(): nicloudsdk.WorkspaceRoleUse,
			},
		})
		require.NoError(t, err)

		inv, root := clitest.New(t,
			"sharing",
			"remove",
			workspace.Name,
			fmt.Sprintf("--user=%s,%s", toRemoveUser1.Username, toRemoveUser2.Username),
		)
		clitest.SetupConfig(t, workspaceOwnerClient, root)

		out := new(bytes.Buffer)
		inv.Stdout = out
		err = inv.WithContext(ctx).Run()
		require.NoError(t, err)

		acl, err := workspaceOwnerClient.WorkspaceACL(inv.Context(), workspace.ID)
		require.NoError(t, err)
		assert.Empty(t, acl.Users)
	})
}
