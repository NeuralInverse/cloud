package cli_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestRename(t *testing.T) {
	t.Parallel()
	logger := testutil.Logger(t)

	client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, AllowWorkspaceRenames: true})
	owner := nicloudtest.CreateFirstUser(t, client)
	member, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
	version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
	workspace := nicloudtest.CreateWorkspace(t, member, template.ID)
	nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	want := nicloudtest.RandomUsername(t)
	inv, root := clitest.New(t, "rename", workspace.Name, want, "--yes")
	clitest.SetupConfig(t, member, root)
	stdout := expecter.NewAttachedToInvocation(t, inv)
	stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
	clitest.Start(t, inv)

	stdout.ExpectMatchContext(ctx, "confirm rename:")
	stdin.WriteLine(workspace.Name)
	stdout.ExpectMatchContext(ctx, "renamed to")

	ws, err := client.Workspace(ctx, workspace.ID)
	assert.NoError(t, err)

	got := ws.Name
	assert.Equal(t, want, got, "workspace name did not change")
}
