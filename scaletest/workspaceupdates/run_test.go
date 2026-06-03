package workspaceupdates_test

import (
	"io"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/scaletest/createusers"
	"github.com/NeuralInverse/cloud/v2/scaletest/workspacebuild"
	"github.com/NeuralInverse/cloud/v2/scaletest/workspaceupdates"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestRun(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t, testutil.WaitSuperLong)

	client := nicloudtest.New(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: true,
	})
	user := nicloudtest.CreateFirstUser(t, client)

	numUsers := 2
	userWorkspaces := 2
	numWorkspaces := numUsers * userWorkspaces

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
	metrics := workspaceupdates.NewMetrics(prometheus.NewRegistry())

	eg, runCtx := errgroup.WithContext(ctx)

	runners := make([]*workspaceupdates.Runner, 0, numUsers)
	for i := range numUsers {
		cfg := workspaceupdates.Config{
			User: createusers.Config{
				OrganizationID: user.OrganizationID,
			},
			Workspace: workspacebuild.Config{
				OrganizationID: user.OrganizationID,
				Request: nicloudsdk.CreateWorkspaceRequest{
					TemplateID: template.ID,
				},
				NoWaitForAgents: true,
			},
			WorkspaceCount:          int64(userWorkspaces),
			DialTimeout:             testutil.WaitMedium,
			WorkspaceUpdatesTimeout: testutil.WaitLong,
			Metrics:                 metrics,
			DialBarrier:             barrier,
		}
		err := cfg.Validate()
		require.NoError(t, err)

		runner := workspaceupdates.NewRunner(client, cfg)
		runners = append(runners, runner)
		eg.Go(func() error {
			return runner.Run(runCtx, strconv.Itoa(i), io.Discard)
		})
	}

	err := eg.Wait()
	require.NoError(t, err)

	users, err := client.Users(ctx, nicloudsdk.UsersRequest{})
	require.NoError(t, err)
	require.Len(t, users.Users, 1+numUsers) // owner + created users

	workspaces, err := client.Workspaces(ctx, nicloudsdk.WorkspaceFilter{})
	require.NoError(t, err)
	require.Len(t, workspaces.Workspaces, numWorkspaces)

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

	for _, runner := range runners {
		metrics := runner.GetMetrics()
		require.Contains(t, metrics, workspaceupdates.WorkspaceUpdatesLatencyMetric)
		require.Len(t, metrics[workspaceupdates.WorkspaceUpdatesLatencyMetric], userWorkspaces)
	}
}
