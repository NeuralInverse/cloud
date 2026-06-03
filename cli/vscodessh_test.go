package cli_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/agent/agenttest"
	agentproto "github.com/NeuralInverse/cloud/v2/agent/proto"
	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloud/workspacestats/workspacestatstest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

// TestVSCodeSSH ensures the agent connects properly with SSH
// and that network information is properly written to the FS.
func TestVSCodeSSH(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitLong)
	dv := nicloudtest.DeploymentValues(t)
	dv.Experiments = []string{string(nicloudsdk.ExperimentWorkspaceUsage)}
	batcher := &workspacestatstest.StatsBatcher{
		LastStats: &agentproto.Stats{},
	}
	admin, store := nicloudtest.NewWithDatabase(t, &nicloudtest.Options{
		DeploymentValues: dv,
		StatsBatcher:     batcher,
	})
	admin.SetLogger(testutil.Logger(t).Named("client"))
	first := nicloudtest.CreateFirstUser(t, admin)
	client, user := nicloudtest.CreateAnotherUser(t, admin, first.OrganizationID)
	r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
		OrganizationID: first.OrganizationID,
		OwnerID:        user.ID,
	}).WithAgent().Do()
	workspace := r.Workspace
	agentToken := r.AgentToken

	user, err := client.User(ctx, nicloudsdk.Me)
	require.NoError(t, err)

	_ = agenttest.New(t, client.URL, agentToken)
	_ = nicloudtest.AwaitWorkspaceAgents(t, client, workspace.ID)

	fs := afero.NewMemMapFs()
	err = afero.WriteFile(fs, "/url", []byte(client.URL.String()), 0o600)
	require.NoError(t, err)
	err = afero.WriteFile(fs, "/token", []byte(client.SessionToken()), 0o600)
	require.NoError(t, err)

	//nolint:revive,staticcheck
	ctx = context.WithValue(ctx, "fs", fs)

	inv, _ := clitest.New(t,
		"vscodessh",
		"--url-file", "/url",
		"--session-token-file", "/token",
		"--network-info-dir", "/net",
		"--log-dir", "/log",
		"--network-info-interval", "25ms",
		fmt.Sprintf("coder-vscode--%s--%s", user.Username, workspace.Name),
	)

	waiter := clitest.StartWithWaiter(t, inv.WithContext(ctx))

	for _, dir := range []string{"/net", "/log"} {
		assert.Eventually(t, func() bool {
			entries, err := afero.ReadDir(fs, dir)
			if err != nil {
				return false
			}
			return len(entries) > 0
		}, testutil.WaitLong, testutil.IntervalFast)
	}
	waiter.Cancel()

	if err := waiter.Wait(); err != nil {
		waiter.RequireIs(context.Canceled)
	}

	require.EqualValues(t, 1, batcher.Called)
	require.EqualValues(t, 1, batcher.LastStats.SessionCountVscode)
}
