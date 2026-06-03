package exptest_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"cdr.dev/slog/v3/sloggers/slogtest"
	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

// This test validates that the scaletest CLI filters out workspaces not owned
// when disable owner workspace access is set.
// This test is in its own package because it mutates a global variable that
// can influence other tests in the same package.
// nolint:paralleltest
func TestScaleTestWorkspaceTraffic_UseHostLogin(t *testing.T) {
	log := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true})
	client := nicloudtest.New(t, &nicloudtest.Options{
		Logger:                   &log,
		IncludeProvisionerDaemon: true,
		DeploymentValues: nicloudtest.DeploymentValues(t, func(dv *nicloudsdk.DeploymentValues) {
			dv.DisableOwnerWorkspaceExec = true
		}),
	})
	owner := nicloudtest.CreateFirstUser(t, client)
	tv := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
	_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, tv.ID)
	tpl := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, tv.ID)
	// Create a workspace owned by a different user
	memberClient, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
	_ = nicloudtest.CreateWorkspace(t, memberClient, tpl.ID, func(cwr *nicloudsdk.CreateWorkspaceRequest) {
		cwr.Name = "scaletest-workspace"
	})

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	// Test without --use-host-login first.g
	inv, root := clitest.New(t, "exp", "scaletest", "workspace-traffic",
		"--template", tpl.Name,
	)
	// nolint:gocritic // We are intentionally testing this as the owner.
	clitest.SetupConfig(t, client, root)
	var stdoutBuf bytes.Buffer
	inv.Stdout = &stdoutBuf

	err := inv.WithContext(ctx).Run()
	require.ErrorContains(t, err, "no scaletest workspaces exist")
	require.Contains(t, stdoutBuf.String(), `1 workspace(s) were skipped`)

	// Test once again with --use-host-login.
	inv, root = clitest.New(t, "exp", "scaletest", "workspace-traffic",
		"--template", tpl.Name,
		"--use-host-login",
	)
	// nolint:gocritic // We are intentionally testing this as the owner.
	clitest.SetupConfig(t, client, root)
	stdoutBuf.Reset()
	inv.Stdout = &stdoutBuf

	err = inv.WithContext(ctx).Run()
	require.ErrorContains(t, err, "no scaletest workspaces exist")
	require.NotContains(t, stdoutBuf.String(), `1 workspace(s) were skipped`)
}
