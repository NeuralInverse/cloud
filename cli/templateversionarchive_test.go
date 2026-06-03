package cli_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestTemplateVersionsArchive(t *testing.T) {
	t.Parallel()
	t.Run("Archive-Unarchive", func(t *testing.T) {
		t.Parallel()
		ownerClient := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, ownerClient)

		client, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		other := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil, func(request *nicloudsdk.CreateTemplateVersionRequest) {
			request.TemplateID = template.ID
		})
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, other.ID)

		// Archive
		inv, root := clitest.New(t, "templates", "versions", "archive", template.Name, other.Name, "-y")
		clitest.SetupConfig(t, client, root)
		w := clitest.StartWithWaiter(t, inv)
		w.RequireSuccess()

		// Verify archived
		ctx := testutil.Context(t, testutil.WaitMedium)
		found, err := client.TemplateVersion(ctx, other.ID)
		require.NoError(t, err)
		require.True(t, found.Archived, "expect archived")

		// Unarchive
		inv, root = clitest.New(t, "templates", "versions", "unarchive", template.Name, other.Name, "-y")
		clitest.SetupConfig(t, client, root)
		w = clitest.StartWithWaiter(t, inv)
		w.RequireSuccess()

		// Verify unarchived
		ctx = testutil.Context(t, testutil.WaitMedium)
		found, err = client.TemplateVersion(ctx, other.ID)
		require.NoError(t, err)
		require.False(t, found.Archived, "expect unarchived")
	})

	t.Run("ArchiveMany", func(t *testing.T) {
		t.Parallel()
		ownerClient := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, ownerClient)

		client, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)

		// Add a failed
		expArchived := map[uuid.UUID]bool{}
		failed := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyFailed,
			ProvisionPlan:  echo.PlanFailed,
			ProvisionInit:  echo.InitComplete,
		}, func(request *nicloudsdk.CreateTemplateVersionRequest) {
			request.TemplateID = template.ID
		})
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, failed.ID)
		expArchived[failed.ID] = true
		// Add unused
		unused := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil, func(request *nicloudsdk.CreateTemplateVersionRequest) {
			request.TemplateID = template.ID
		})
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, unused.ID)
		expArchived[unused.ID] = true

		// Archive all unused versions
		inv, root := clitest.New(t, "templates", "archive", template.Name, "-y", "--all")
		clitest.SetupConfig(t, client, root)
		w := clitest.StartWithWaiter(t, inv)
		w.RequireSuccess()

		ctx := testutil.Context(t, testutil.WaitMedium)
		all, err := client.TemplateVersionsByTemplate(ctx, nicloudsdk.TemplateVersionsByTemplateRequest{
			TemplateID:      template.ID,
			IncludeArchived: true,
		})
		require.NoError(t, err, "query all versions")
		for _, v := range all {
			if _, ok := expArchived[v.ID]; ok {
				require.True(t, v.Archived, "expect archived")
				delete(expArchived, v.ID)
			} else {
				require.False(t, v.Archived, "expect unarchived")
			}
		}
		require.Len(t, expArchived, 0, "expect all archived")
	})
}
