package cli_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestRestart(t *testing.T) {
	t.Parallel()

	echoResponses := func() *echo.Responses {
		return prepareEchoResponses([]*proto.RichParameter{
			{
				Name:        ephemeralParameterName,
				Description: ephemeralParameterDescription,
				Mutable:     true,
				Ephemeral:   true,
			},
		})
	}

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		inv, root := clitest.New(t, "restart", workspace.Name, "--yes")
		clitest.SetupConfig(t, member, root)

		stdout := expecter.NewAttachedToInvocation(t, inv)

		done := make(chan error, 1)
		go func() {
			done <- inv.WithContext(ctx).Run()
		}()
		stdout.ExpectMatchContext(ctx, "Stopping workspace")
		stdout.ExpectMatchContext(ctx, "Starting workspace")
		stdout.ExpectMatchContext(ctx, "workspace has been restarted")

		err := <-done
		require.NoError(t, err, "execute failed")
	})

	t.Run("PromptEphemeralParameters", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, memberUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, echoResponses())
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID, func(request *nicloudsdk.CreateTemplateRequest) {
			request.UseClassicParameterFlow = ptr.Ref(true) // TODO: Remove when dynamic parameters prompt missing ephemeral parameters.
		})
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID, func(request *nicloudsdk.CreateWorkspaceRequest) {
			request.RichParameterValues = []nicloudsdk.WorkspaceBuildParameter{
				{Name: ephemeralParameterName, Value: "placeholder"},
			}
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		inv, root := clitest.New(t, "restart", workspace.Name, "--prompt-ephemeral-parameters")
		clitest.SetupConfig(t, member, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()

		ctx := testutil.Context(t, testutil.WaitShort)
		matches := []string{
			ephemeralParameterDescription, ephemeralParameterValue,
			"Restart workspace?", "yes",
			"Stopping workspace", "",
			"Starting workspace", "",
			"workspace has been restarted", "",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)

			if value != "" {
				stdin.WriteLine(value)
			}
		}
		<-doneChan

		// Verify if build option is set
		workspace, err := client.WorkspaceByOwnerAndName(ctx, memberUser.ID.String(), workspace.Name, nicloudsdk.WorkspaceOptions{})
		require.NoError(t, err)
		actualParameters, err := client.WorkspaceBuildParameters(ctx, workspace.LatestBuild.ID)
		require.NoError(t, err)
		require.Contains(t, actualParameters, nicloudsdk.WorkspaceBuildParameter{
			Name:  ephemeralParameterName,
			Value: ephemeralParameterValue,
		})
	})

	t.Run("EphemeralParameterFlags", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, memberUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, echoResponses())
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID, func(request *nicloudsdk.CreateWorkspaceRequest) {
			request.RichParameterValues = []nicloudsdk.WorkspaceBuildParameter{
				{Name: ephemeralParameterName, Value: "placeholder"},
			}
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		inv, root := clitest.New(t, "restart", workspace.Name,
			"--ephemeral-parameter", fmt.Sprintf("%s=%s", ephemeralParameterName, ephemeralParameterValue))
		clitest.SetupConfig(t, member, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()

		ctx := testutil.Context(t, testutil.WaitShort)
		matches := []string{
			"Restart workspace?", "yes",
			"Stopping workspace", "",
			"Starting workspace", "",
			"workspace has been restarted", "",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)

			if value != "" {
				stdin.WriteLine(value)
			}
		}
		<-doneChan

		// Verify if build option is set
		workspace, err := client.WorkspaceByOwnerAndName(ctx, memberUser.ID.String(), workspace.Name, nicloudsdk.WorkspaceOptions{})
		require.NoError(t, err)
		actualParameters, err := client.WorkspaceBuildParameters(ctx, workspace.LatestBuild.ID)
		require.NoError(t, err)
		require.Contains(t, actualParameters, nicloudsdk.WorkspaceBuildParameter{
			Name:  ephemeralParameterName,
			Value: ephemeralParameterValue,
		})
	})

	t.Run("with deprecated build-options flag", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, memberUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, echoResponses())
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID, func(request *nicloudsdk.CreateTemplateRequest) {
			request.UseClassicParameterFlow = ptr.Ref(true) // TODO: Remove when dynamic parameters prompts missing ephemeral parameters
		})
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID, func(request *nicloudsdk.CreateWorkspaceRequest) {
			request.RichParameterValues = []nicloudsdk.WorkspaceBuildParameter{
				{Name: ephemeralParameterName, Value: "placeholder"},
			}
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		inv, root := clitest.New(t, "restart", workspace.Name, "--build-options")
		clitest.SetupConfig(t, member, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()

		ctx := testutil.Context(t, testutil.WaitShort)
		matches := []string{
			ephemeralParameterDescription, ephemeralParameterValue,
			"Restart workspace?", "yes",
			"Stopping workspace", "",
			"Starting workspace", "",
			"workspace has been restarted", "",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)

			if value != "" {
				stdin.WriteLine(value)
			}
		}
		<-doneChan

		// Verify if build option is set
		workspace, err := client.WorkspaceByOwnerAndName(ctx, memberUser.ID.String(), workspace.Name, nicloudsdk.WorkspaceOptions{})
		require.NoError(t, err)
		actualParameters, err := client.WorkspaceBuildParameters(ctx, workspace.LatestBuild.ID)
		require.NoError(t, err)
		require.Contains(t, actualParameters, nicloudsdk.WorkspaceBuildParameter{
			Name:  ephemeralParameterName,
			Value: ephemeralParameterValue,
		})
	})

	t.Run("with deprecated build-option flag", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, memberUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, echoResponses())
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID, func(request *nicloudsdk.CreateWorkspaceRequest) {
			request.RichParameterValues = []nicloudsdk.WorkspaceBuildParameter{
				{Name: ephemeralParameterName, Value: "placeholder"},
			}
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		inv, root := clitest.New(t, "restart", workspace.Name,
			"--build-option", fmt.Sprintf("%s=%s", ephemeralParameterName, ephemeralParameterValue))
		clitest.SetupConfig(t, member, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()

		ctx := testutil.Context(t, testutil.WaitShort)
		matches := []string{
			"Restart workspace?", "yes",
			"Stopping workspace", "",
			"Starting workspace", "",
			"workspace has been restarted", "",
		}
		for i := 0; i < len(matches); i += 2 {
			match := matches[i]
			value := matches[i+1]
			stdout.ExpectMatchContext(ctx, match)

			if value != "" {
				stdin.WriteLine(value)
			}
		}
		<-doneChan

		// Verify if build option is set
		workspace, err := client.WorkspaceByOwnerAndName(ctx, memberUser.ID.String(), workspace.Name, nicloudsdk.WorkspaceOptions{})
		require.NoError(t, err)
		actualParameters, err := client.WorkspaceBuildParameters(ctx, workspace.LatestBuild.ID)
		require.NoError(t, err)
		require.Contains(t, actualParameters, nicloudsdk.WorkspaceBuildParameter{
			Name:  ephemeralParameterName,
			Value: ephemeralParameterValue,
		})
	})
}

func TestRestartWithParameters(t *testing.T) {
	t.Parallel()

	echoResponses := func() *echo.Responses {
		return &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionGraph: []*proto.Response{
				{
					Type: &proto.Response_Graph{
						Graph: &proto.GraphComplete{
							Parameters: []*proto.RichParameter{
								{
									Name:        immutableParameterName,
									Description: immutableParameterDescription,
									Required:    true,
								},
							},
						},
					},
				},
			},
			ProvisionApply: echo.ApplyComplete,
		}
	}

	t.Run("DoNotAskForImmutables", func(t *testing.T) {
		t.Parallel()

		// Create the workspace
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, echoResponses())
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID, func(cwr *nicloudsdk.CreateWorkspaceRequest) {
			cwr.RichParameterValues = []nicloudsdk.WorkspaceBuildParameter{
				{
					Name:  immutableParameterName,
					Value: immutableParameterValue,
				},
			}
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		// Restart the workspace again
		inv, root := clitest.New(t, "restart", workspace.Name, "-y")
		clitest.SetupConfig(t, member, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()
		ctx := testutil.Context(t, testutil.WaitShort)

		stdout.ExpectMatchContext(ctx, "workspace has been restarted")
		<-doneChan

		// Verify if immutable parameter is set
		workspace, err := client.WorkspaceByOwnerAndName(ctx, workspace.OwnerName, workspace.Name, nicloudsdk.WorkspaceOptions{})
		require.NoError(t, err)
		actualParameters, err := client.WorkspaceBuildParameters(ctx, workspace.LatestBuild.ID)
		require.NoError(t, err)
		require.Contains(t, actualParameters, nicloudsdk.WorkspaceBuildParameter{
			Name:  immutableParameterName,
			Value: immutableParameterValue,
		})
	})

	t.Run("AlwaysPrompt", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		// Create the workspace
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, mutableParamsResponse())
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID, func(cwr *nicloudsdk.CreateWorkspaceRequest) {
			cwr.RichParameterValues = []nicloudsdk.WorkspaceBuildParameter{
				{
					Name:  mutableParameterName,
					Value: mutableParameterValue,
				},
			}
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		inv, root := clitest.New(t, "restart", workspace.Name, "-y", "--always-prompt")
		clitest.SetupConfig(t, member, root)
		doneChan := make(chan struct{})
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)
		go func() {
			defer close(doneChan)
			err := inv.Run()
			assert.NoError(t, err)
		}()
		ctx := testutil.Context(t, testutil.WaitShort)

		// We should be prompted for the parameters again.
		newValue := "xyz"
		stdout.ExpectMatchContext(ctx, mutableParameterName)
		stdin.WriteLine(newValue)
		stdout.ExpectMatchContext(ctx, "workspace has been restarted")
		<-doneChan

		// Verify that the updated values are persisted.
		workspace, err := client.WorkspaceByOwnerAndName(ctx, workspace.OwnerName, workspace.Name, nicloudsdk.WorkspaceOptions{})
		require.NoError(t, err)
		actualParameters, err := client.WorkspaceBuildParameters(ctx, workspace.LatestBuild.ID)
		require.NoError(t, err)
		require.Contains(t, actualParameters, nicloudsdk.WorkspaceBuildParameter{
			Name:  mutableParameterName,
			Value: newValue,
		})
	})
}
