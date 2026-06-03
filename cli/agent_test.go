package cli_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/agent"
	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestWorkspaceAgent(t *testing.T) {
	t.Parallel()

	t.Run("LogDirectory", func(t *testing.T) {
		t.Parallel()

		client, db := nicloudtest.NewWithDatabase(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
			OrganizationID: user.OrganizationID,
			OwnerID:        user.UserID,
		}).
			WithAgent().
			Do()
		logDir := t.TempDir()
		inv, _ := clitest.New(t,
			"agent",
			"--auth", "token",
			"--agent-token", r.AgentToken,
			"--agent-url", client.URL.String(),
			"--log-dir", logDir,
			"--socket-path", testutil.AgentSocketPath(t),
		)

		clitest.Start(t, inv)

		nicloudtest.AwaitWorkspaceAgents(t, client, r.Workspace.ID)

		require.Eventually(t, func() bool {
			info, err := os.Stat(filepath.Join(logDir, "coder-agent.log"))
			if err != nil {
				return false
			}
			return info.Size() > 0
		}, testutil.WaitLong, testutil.IntervalMedium)
	})

	t.Run("PostStartup", func(t *testing.T) {
		t.Parallel()

		client, db := nicloudtest.NewWithDatabase(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
			OrganizationID: user.OrganizationID,
			OwnerID:        user.UserID,
		}).WithAgent().Do()

		logDir := t.TempDir()
		inv, _ := clitest.New(t,
			"agent",
			"--auth", "token",
			"--agent-token", r.AgentToken,
			"--agent-url", client.URL.String(),
			"--log-dir", logDir,
			"--socket-path", testutil.AgentSocketPath(t),
		)
		// Set the subsystems for the agent.
		inv.Environ.Set(agent.EnvAgentSubsystem, fmt.Sprintf("%s,%s", nicloudsdk.AgentSubsystemExectrace, nicloudsdk.AgentSubsystemEnvbox))

		clitest.Start(t, inv)

		resources := nicloudtest.NewWorkspaceAgentWaiter(t, client, r.Workspace.ID).
			MatchResources(matchAgentWithSubsystems).Wait()
		require.Len(t, resources, 1)
		require.Len(t, resources[0].Agents, 1)
		require.Len(t, resources[0].Agents[0].Subsystems, 2)
		// Sorted
		require.Equal(t, nicloudsdk.AgentSubsystemEnvbox, resources[0].Agents[0].Subsystems[0])
		require.Equal(t, nicloudsdk.AgentSubsystemExectrace, resources[0].Agents[0].Subsystems[1])
	})
	t.Run("Headers&DERPHeaders", func(t *testing.T) {
		t.Parallel()

		// Create a nicloud API instance the hard way since we need to change the
		// handler to inject our custom /derp handler.
		dv := nicloudtest.DeploymentValues(t)
		dv.DERP.Config.BlockDirect = true
		setHandler, cancelFunc, serverURL, newOptions := nicloudtest.NewOptions(t, &nicloudtest.Options{
			DeploymentValues: dv,
		})

		// We set the handler after server creation for the access URL.
		niAPI := nicloud.New(newOptions)
		setHandler(niAPI.RootHandler)
		provisionerCloser := nicloudtest.NewProvisionerDaemon(t, niAPI)
		t.Cleanup(func() {
			_ = provisionerCloser.Close()
		})
		client := nicloudsdk.New(serverURL, nicloudsdk.WithHTTPClient(nicloudtest.NewIsolatedHTTPClient(serverURL)))
		t.Cleanup(func() {
			cancelFunc()
			_ = provisionerCloser.Close()
			_ = niAPI.Close()
			client.HTTPClient.CloseIdleConnections()
		})

		var (
			admin              = nicloudtest.CreateFirstUser(t, client)
			member, memberUser = nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID)
			called             atomic.Int64
			derpCalled         atomic.Int64
		)

		setHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ignore client requests
			if r.Header.Get("X-Testing") == "agent" {
				assert.Equal(t, "Ethan was Here!", r.Header.Get("Cool-Header"))
				assert.Equal(t, "very-wow-"+client.URL.String(), r.Header.Get("X-Process-Testing"))
				assert.Equal(t, "more-wow", r.Header.Get("X-Process-Testing2"))
				if strings.HasPrefix(r.URL.Path, "/derp") {
					derpCalled.Add(1)
				} else {
					called.Add(1)
				}
			}
			niAPI.RootHandler.ServeHTTP(w, r)
		}))
		r := dbfake.WorkspaceBuild(t, niAPI.Database, database.WorkspaceTable{
			OrganizationID: memberUser.OrganizationIDs[0],
			OwnerID:        memberUser.ID,
		}).WithAgent().Do()

		niURLEnv := "$NEURALINVERSE_URL"
		if runtime.GOOS == "windows" {
			niURLEnv = "%NEURALINVERSE_URL%"
		}

		logDir := t.TempDir()
		agentInv, _ := clitest.New(t,
			"agent",
			"--auth", "token",
			"--agent-token", r.AgentToken,
			"--agent-url", client.URL.String(),
			"--log-dir", logDir,
			"--agent-header", "X-Testing=agent",
			"--agent-header", "Cool-Header=Ethan was Here!",
			"--agent-header-command", "printf X-Process-Testing=very-wow-"+niURLEnv+"'\\r\\n'X-Process-Testing2=more-wow",
			"--socket-path", testutil.AgentSocketPath(t),
		)
		clitest.Start(t, agentInv)
		nicloudtest.NewWorkspaceAgentWaiter(t, client, r.Workspace.ID).
			MatchResources(matchAgentWithVersion).Wait()

		ctx := testutil.Context(t, testutil.WaitLong)
		clientInv, root := clitest.New(t,
			"-v",
			"--no-feature-warning",
			"--no-version-warning",
			"ping", r.Workspace.Name,
			"-n", "1",
		)
		clitest.SetupConfig(t, member, root)
		err := clientInv.WithContext(ctx).Run()
		require.NoError(t, err)

		require.Greater(t, called.Load(), int64(0), "expected nicloud to be reached with custom headers")
		require.Greater(t, derpCalled.Load(), int64(0), "expected /derp to be called with custom headers")
	})

	t.Run("DisabledServers", func(t *testing.T) {
		t.Parallel()

		client, db := nicloudtest.NewWithDatabase(t, nil)
		user := nicloudtest.CreateFirstUser(t, client)
		r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
			OrganizationID: user.OrganizationID,
			OwnerID:        user.UserID,
		}).WithAgent().Do()

		logDir := t.TempDir()
		inv, _ := clitest.New(t,
			"agent",
			"--auth", "token",
			"--agent-token", r.AgentToken,
			"--agent-url", client.URL.String(),
			"--log-dir", logDir,
			"--pprof-address", "",
			"--prometheus-address", "",
			"--debug-address", "",
			"--socket-path", testutil.AgentSocketPath(t),
		)

		clitest.Start(t, inv)

		// Verify the agent is connected and working.
		resources := nicloudtest.NewWorkspaceAgentWaiter(t, client, r.Workspace.ID).
			MatchResources(matchAgentWithVersion).Wait()
		require.Len(t, resources, 1)
		require.Len(t, resources[0].Agents, 1)
		require.NotEmpty(t, resources[0].Agents[0].Version)

		// Verify the servers are not listening by checking the log for disabled
		// messages.
		require.Eventually(t, func() bool {
			logContent, err := os.ReadFile(filepath.Join(logDir, "coder-agent.log"))
			if err != nil {
				return false
			}
			logStr := string(logContent)
			return strings.Contains(logStr, "pprof address is empty, disabling pprof server") &&
				strings.Contains(logStr, "prometheus address is empty, disabling prometheus server") &&
				strings.Contains(logStr, "debug address is empty, disabling debug server")
		}, testutil.WaitLong, testutil.IntervalMedium)
	})
}

func matchAgentWithVersion(rs []nicloudsdk.WorkspaceResource) bool {
	if len(rs) < 1 {
		return false
	}
	if len(rs[0].Agents) < 1 {
		return false
	}
	if rs[0].Agents[0].Version == "" {
		return false
	}
	return true
}

func matchAgentWithSubsystems(rs []nicloudsdk.WorkspaceResource) bool {
	if len(rs) < 1 {
		return false
	}
	if len(rs[0].Agents) < 1 {
		return false
	}
	if len(rs[0].Agents[0].Subsystems) < 1 {
		return false
	}
	return true
}
