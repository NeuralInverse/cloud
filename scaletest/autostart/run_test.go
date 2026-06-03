package autostart_test

import (
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/scaletest/autostart"
	"github.com/NeuralInverse/cloud/v2/scaletest/createusers"
	"github.com/NeuralInverse/cloud/v2/scaletest/loadtestutil"
	"github.com/NeuralInverse/cloud/v2/scaletest/workspacebuild"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestRun(t *testing.T) {
	t.Parallel()
	numUsers := 2
	autoStartDelay := 2 * time.Minute

	// Faking a workspace autostart schedule start time at the nicloud level
	// is difficult and error-prone. This test verifies the setup phase only
	// (creating workspaces, stopping them, and configuring autostart schedules).
	t.Skip("This test takes several minutes to run, and is intended as a manual regression test")

	ctx := testutil.Context(t, time.Minute*3)

	client := nicloudtest.New(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: true,
		AutobuildTicker:          time.NewTicker(time.Second * 1).C,
		DeploymentValues: nicloudtest.DeploymentValues(t, func(dv *nicloudsdk.DeploymentValues) {
			dv.Experiments = []string{string(nicloudsdk.ExperimentWorkspaceBuildUpdates)}
		}),
	})
	user := nicloudtest.CreateFirstUser(t, client)

	authToken := uuid.NewString()
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse:         echo.ParseComplete,
		ProvisionPlan: echo.PlanComplete,
		ProvisionGraph: []*proto.Response{
			{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Resources: []*proto.Resource{
							{
								Name: "example",
								Type: "aws_instance",
								Agents: []*proto.Agent{
									{
										Id:   uuid.NewString(),
										Name: "agent",
										Auth: &proto.Agent_Token{
											Token: authToken,
										},
										Apps: []*proto.App{},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

	barrier := new(sync.WaitGroup)
	barrier.Add(numUsers)

	// Pre-create channels for each workspace keyed by deterministic name.
	workspaceChannels := make(map[string]chan nicloudsdk.WorkspaceBuildUpdate)
	for i := range numUsers {
		id := strconv.Itoa(i)
		workspaceName := loadtestutil.GenerateDeterministicWorkspaceName(id)
		workspaceChannels[workspaceName] = make(chan nicloudsdk.WorkspaceBuildUpdate, 16)
	}

	// Start watching all workspace builds.
	decoder, err := client.WatchAllWorkspaceBuilds(ctx)
	require.NoError(t, err)
	defer decoder.Close()

	// Start the dispatcher goroutine.
	go func() {
		for update := range decoder.Chan() {
			if ch, ok := workspaceChannels[update.WorkspaceName]; ok {
				select {
				case ch <- update:
				case <-ctx.Done():
					return
				}
			}
		}
		for _, ch := range workspaceChannels {
			close(ch)
		}
	}()

	eg, runCtx := errgroup.WithContext(ctx)

	runners := make([]*autostart.Runner, 0, numUsers)
	for i := range numUsers {
		id := strconv.Itoa(i)
		workspaceName := loadtestutil.GenerateDeterministicWorkspaceName(id)
		cfg := autostart.Config{
			User: createusers.Config{
				OrganizationID: user.OrganizationID,
			},
			Workspace: workspacebuild.Config{
				OrganizationID: user.OrganizationID,
				Request: nicloudsdk.CreateWorkspaceRequest{
					TemplateID: template.ID,
					Name:       workspaceName,
				},
				NoWaitForAgents: true,
			},
			WorkspaceJobTimeout: testutil.WaitMedium,
			AutostartDelay:      autoStartDelay,
			SetupBarrier:        barrier,
			BuildUpdates:        workspaceChannels[workspaceName],
		}
		err := cfg.Validate()
		require.NoError(t, err)

		runner := autostart.NewRunner(client, cfg)
		runners = append(runners, runner)
		eg.Go(func() error {
			return runner.Run(runCtx, strconv.Itoa(i), io.Discard)
		})
	}

	err = eg.Wait()
	require.NoError(t, err)

	users, err := client.Users(ctx, nicloudsdk.UsersRequest{})
	require.NoError(t, err)
	require.Len(t, users.Users, 1+numUsers) // owner + created users

	workspaces, err := client.Workspaces(ctx, nicloudsdk.WorkspaceFilter{})
	require.NoError(t, err)
	require.Len(t, workspaces.Workspaces, numUsers) // one workspace per user

	// Verify that workspaces have autostart schedules set and are stopped
	// (the test exits after configuring autostart, before it triggers).
	for _, workspace := range workspaces.Workspaces {
		require.NotNil(t, workspace.AutostartSchedule)
		require.Equal(t, nicloudsdk.WorkspaceTransitionStop, workspace.LatestBuild.Transition)
		require.Equal(t, nicloudsdk.ProvisionerJobSucceeded, workspace.LatestBuild.Job.Status)
	}

	cleanupEg, cleanupCtx := errgroup.WithContext(ctx)
	for i, runner := range runners {
		cleanupEg.Go(func() error {
			return runner.Cleanup(cleanupCtx, strconv.Itoa(i), io.Discard)
		})
	}
	err = cleanupEg.Wait()
	require.NoError(t, err)

	workspaces, err = client.Workspaces(ctx, nicloudsdk.WorkspaceFilter{})
	require.NoError(t, err)
	require.Len(t, workspaces.Workspaces, 0)

	users, err = client.Users(ctx, nicloudsdk.UsersRequest{})
	require.NoError(t, err)
	require.Len(t, users.Users, 1) // owner
}
