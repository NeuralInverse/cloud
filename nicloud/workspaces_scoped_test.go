package nicloud_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

// TestCompositeWorkspaceScopes verifies that the composite
// coder:workspaces.* scopes grant the permissions needed for
// workspace lifecycle operations when used on scoped API tokens.
func TestCompositeWorkspaceScopes(t *testing.T) {
	t.Parallel()

	// setupWorkspace creates a server with a provisioner daemon, an
	// admin user, a template, and a workspace. It returns the admin
	// client and the workspace so sub-tests can create scoped tokens
	// and act on them.
	type setupResult struct {
		adminClient *nicloudsdk.Client
		workspace   nicloudsdk.Workspace
	}
	setup := func(t *testing.T) setupResult {
		t.Helper()
		client := nicloudtest.New(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		})
		firstUser := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, firstUser.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionPlan:  echo.PlanComplete,
			ProvisionApply: echo.ApplyComplete,
			ProvisionGraph: echo.GraphComplete,
		})
		template := nicloudtest.CreateTemplate(t, client, firstUser.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		return setupResult{
			adminClient: client,
			workspace:   workspace,
		}
	}

	// scopedClient creates an API token restricted to the given scopes
	// and returns a new client authenticated with that token.
	scopedClient := func(t *testing.T, adminClient *nicloudsdk.Client, scopes []nicloudsdk.APIKeyScope) *nicloudsdk.Client {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), testutil.WaitShort)
		defer cancel()

		resp, err := adminClient.CreateToken(ctx, nicloudsdk.Me, nicloudsdk.CreateTokenRequest{
			Scopes: scopes,
		})
		require.NoError(t, err, "creating scoped token")

		scoped := nicloudsdk.New(
			adminClient.URL,
			nicloudsdk.WithSessionToken(resp.Key),
			nicloudsdk.WithHTTPClient(nicloudtest.NewIsolatedHTTPClient(adminClient.URL)),
		)
		t.Cleanup(func() { scoped.HTTPClient.CloseIdleConnections() })
		return scoped
	}

	// coder:workspaces.create — token should be able to create a
	// workspace via POST /users/{user}/workspaces.
	t.Run("WorkspacesCreate", func(t *testing.T) {
		t.Parallel()
		s := setup(t)

		scoped := scopedClient(t, s.adminClient, []nicloudsdk.APIKeyScope{
			nicloudsdk.APIKeyScopeCoderWorkspacesCreate,
		})

		ctx, cancel := context.WithTimeout(t.Context(), testutil.WaitLong)
		defer cancel()

		// List workspaces (requires workspace:read, included in the
		// composite scope).
		workspaces, err := scoped.Workspaces(ctx, nicloudsdk.WorkspaceFilter{})
		require.NoError(t, err, "listing workspaces with coder:workspaces.create scope")
		require.NotEmpty(t, workspaces.Workspaces, "should see at least the existing workspace")

		_, err = scoped.CreateUserWorkspace(ctx, nicloudsdk.Me, nicloudsdk.CreateWorkspaceRequest{
			TemplateID: s.workspace.TemplateID,
			Name:       nicloudtest.RandomUsername(t),
		})
		require.NoError(t, err, "creating workspace with coder:workspaces.create scope")
	})

	// coder:workspaces.operate — token should be able to read and
	// update workspace metadata.
	t.Run("WorkspacesOperate", func(t *testing.T) {
		t.Parallel()
		s := setup(t)

		scoped := scopedClient(t, s.adminClient, []nicloudsdk.APIKeyScope{
			nicloudsdk.APIKeyScopeCoderWorkspacesOperate,
		})

		ctx, cancel := context.WithTimeout(t.Context(), testutil.WaitLong)
		defer cancel()

		// Read the workspace by ID (requires workspace:read).
		ws, err := scoped.Workspace(ctx, s.workspace.ID)
		require.NoError(t, err, "reading workspace with coder:workspaces.operate scope")
		require.Equal(t, s.workspace.ID, ws.ID)

		// Update the workspace metadata (requires workspace:update). This goes
		// through the PATCH /workspaces/{workspace} endpoint.
		err = scoped.UpdateWorkspaceTTL(ctx, s.workspace.ID, nicloudsdk.UpdateWorkspaceTTLRequest{
			TTLMillis: ptr.Ref[int64]((time.Hour).Milliseconds()),
		})
		require.NoError(t, err, "updating workspace with coder:workspaces.operate scope")

		// Trigger a start build (requires workspace:update). This goes
		// through POST /workspaces/{workspace}/builds.
		started, err := scoped.CreateWorkspaceBuild(ctx, s.workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: ws.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		require.NoError(t, err, "starting workspace with coder:workspaces.operate scope")
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, scoped, started.ID)

		_, err = scoped.CreateWorkspaceBuild(ctx, s.workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: ws.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStop,
		})
		require.NoError(t, err, "starting workspace with coder:workspaces.operate scope")

		// Verify we cannot create a new workspace — the operate scope
		// should not include workspace:create or template:read/use.
		_, err = scoped.CreateUserWorkspace(ctx, nicloudsdk.Me, nicloudsdk.CreateWorkspaceRequest{
			TemplateID: s.workspace.TemplateID,
			Name:       nicloudtest.RandomUsername(t),
		})
		require.Error(t, err, "creating workspace should fail with coder:workspaces.operate scope")
	})

	// coder:workspaces.delete — token should be able to read
	// workspaces and trigger a delete build.
	t.Run("WorkspacesDelete", func(t *testing.T) {
		t.Parallel()
		s := setup(t)

		scoped := scopedClient(t, s.adminClient, []nicloudsdk.APIKeyScope{
			nicloudsdk.APIKeyScopeCoderWorkspacesDelete,
		})

		ctx, cancel := context.WithTimeout(t.Context(), testutil.WaitLong)
		defer cancel()

		// Read the workspace by ID (requires workspace:read).
		ws, err := scoped.Workspace(ctx, s.workspace.ID)
		require.NoError(t, err, "reading workspace with coder:workspaces.delete scope")
		require.Equal(t, s.workspace.ID, ws.ID)

		// Delete the workspace via a delete transition build.
		_, err = scoped.CreateWorkspaceBuild(ctx, s.workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: ws.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionDelete,
		})
		require.NoError(t, err, "deleting workspace with coder:workspaces.delete scope")
	})
}
