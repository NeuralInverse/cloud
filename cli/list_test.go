package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestList(t *testing.T) {
	t.Parallel()
	t.Run("Single", func(t *testing.T) {
		t.Parallel()
		client, db := nicloudtest.NewWithDatabase(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		member, memberUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		// setup template
		r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
			OrganizationID: owner.OrganizationID,
			OwnerID:        memberUser.ID,
		}).WithAgent().Do()

		inv, root := clitest.New(t, "ls")
		clitest.SetupConfig(t, member, root)
		stdout := expecter.NewAttachedToInvocation(t, inv)

		ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancelFunc()
		done := make(chan any)
		go func() {
			errC := inv.WithContext(ctx).Run()
			assert.NoError(t, errC)
			close(done)
		}()
		stdout.ExpectMatchContext(ctx, r.Workspace.Name)
		stdout.ExpectMatchContext(ctx, "Started")
		cancelFunc()
		<-done
	})

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()
		client, db := nicloudtest.NewWithDatabase(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		member, memberUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		_ = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
			OrganizationID: owner.OrganizationID,
			OwnerID:        memberUser.ID,
		}).WithAgent().Do()

		inv, root := clitest.New(t, "list", "--output=json")
		clitest.SetupConfig(t, member, root)

		ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancelFunc()

		out := bytes.NewBuffer(nil)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		var workspaces []nicloudsdk.Workspace
		require.NoError(t, json.Unmarshal(out.Bytes(), &workspaces))
		require.Len(t, workspaces, 1)
	})

	t.Run("NoWorkspacesJSON", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		member, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)

		inv, root := clitest.New(t, "list", "--output=json")
		clitest.SetupConfig(t, member, root)

		ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancelFunc()

		stdout := bytes.NewBuffer(nil)
		stderr := bytes.NewBuffer(nil)
		inv.Stdout = stdout
		inv.Stderr = stderr
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		var workspaces []nicloudsdk.Workspace
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &workspaces))
		require.Len(t, workspaces, 0)

		require.Len(t, stderr.Bytes(), 0)
	})

	t.Run("SharedWorkspaces", func(t *testing.T) {
		t.Parallel()

		var (
			client, db           = nicloudtest.NewWithDatabase(t, nil)
			orgOwner             = nicloudtest.CreateFirstUser(t, client)
			memberClient, member = nicloudtest.CreateAnotherUser(t, client, orgOwner.OrganizationID, rbac.ScopedRoleOrgAuditor(orgOwner.OrganizationID))
			sharedWorkspace      = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				Name:           "wibble",
				OwnerID:        orgOwner.UserID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace
			_ = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
				Name:           "wobble",
				OwnerID:        orgOwner.UserID,
				OrganizationID: orgOwner.OrganizationID,
			}).Do().Workspace
		)

		ctx := testutil.Context(t, testutil.WaitMedium)

		client.UpdateWorkspaceACL(ctx, sharedWorkspace.ID, nicloudsdk.UpdateWorkspaceACL{
			UserRoles: map[string]nicloudsdk.WorkspaceRole{
				member.ID.String(): nicloudsdk.WorkspaceRoleUse,
			},
		})

		inv, root := clitest.New(t, "list", "--shared-with-me", "--output=json")
		clitest.SetupConfig(t, memberClient, root)

		stdout := new(bytes.Buffer)
		inv.Stdout = stdout
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		var workspaces []nicloudsdk.Workspace
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &workspaces))
		require.Len(t, workspaces, 1)
		require.Equal(t, sharedWorkspace.ID, workspaces[0].ID)
	})
}
