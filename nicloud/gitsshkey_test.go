package nicloud_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/gitsshkey"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/agentsdk"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestGitSSHKey(t *testing.T) {
	t.Parallel()
	t.Run("None", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, nil)
		res := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		key, err := client.GitSSHKey(ctx, res.UserID.String())
		require.NoError(t, err)
		require.NotEmpty(t, key.PublicKey)
	})
	t.Run("Ed25519", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{
			SSHKeygenAlgorithm: gitsshkey.AlgorithmEd25519,
		})
		res := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		key, err := client.GitSSHKey(ctx, res.UserID.String())
		require.NoError(t, err)
		require.NotEmpty(t, key.PublicKey)
	})
	t.Run("ECDSA", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{
			SSHKeygenAlgorithm: gitsshkey.AlgorithmECDSA,
		})
		res := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		key, err := client.GitSSHKey(ctx, res.UserID.String())
		require.NoError(t, err)
		require.NotEmpty(t, key.PublicKey)
	})
	t.Run("RSA4096", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{
			SSHKeygenAlgorithm: gitsshkey.AlgorithmRSA4096,
		})
		res := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		key, err := client.GitSSHKey(ctx, res.UserID.String())
		require.NoError(t, err)
		require.NotEmpty(t, key.PublicKey)
	})
	t.Run("Regenerate", func(t *testing.T) {
		t.Parallel()
		auditor := audit.NewMock()
		client := nicloudtest.New(t, &nicloudtest.Options{
			SSHKeygenAlgorithm: gitsshkey.AlgorithmEd25519,
			Auditor:            auditor,
		})
		res := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		key1, err := client.GitSSHKey(ctx, res.UserID.String())
		require.NoError(t, err)
		require.NotEmpty(t, key1.PublicKey)
		key2, err := client.RegenerateGitSSHKey(ctx, res.UserID.String())
		require.NoError(t, err)
		require.GreaterOrEqual(t, key2.UpdatedAt, key1.UpdatedAt)
		require.NotEmpty(t, key2.PublicKey)

		require.Len(t, auditor.AuditLogs(), 2)
		assert.Equal(t, database.AuditActionWrite, auditor.AuditLogs()[1].Action)
	})
}

func TestAgentGitSSHKey(t *testing.T) {
	t.Parallel()

	client := nicloudtest.New(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: true,
	})
	user := nicloudtest.CreateFirstUser(t, client)
	authToken := uuid.NewString()
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse:          echo.ParseComplete,
		ProvisionPlan:  echo.PlanComplete,
		ProvisionGraph: echo.ProvisionGraphWithAgent(authToken),
	})
	project := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	workspace := nicloudtest.CreateWorkspace(t, client, project.ID)
	nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

	agentClient := agentsdk.New(client.URL, agentsdk.WithFixedToken(authToken))

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	agentKey, err := agentClient.GitSSHKey(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, agentKey.PrivateKey)
}

func TestAgentGitSSHKey_APIKeyScopes(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		apiKeyScope string
		expectError bool
	}{
		{apiKeyScope: "all", expectError: false},
		{apiKeyScope: "no_user_data", expectError: true},
	} {
		t.Run(tt.apiKeyScope, func(t *testing.T) {
			t.Parallel()

			client := nicloudtest.New(t, &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			})
			user := nicloudtest.CreateFirstUser(t, client)
			authToken := uuid.NewString()
			version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
				Parse:          echo.ParseComplete,
				ProvisionPlan:  echo.PlanComplete,
				ProvisionGraph: echo.ProvisionGraphWithAgentAndAPIKeyScope(authToken, tt.apiKeyScope),
			})
			project := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
			nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
			workspace := nicloudtest.CreateWorkspace(t, client, project.ID)
			nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

			agentClient := agentsdk.New(client.URL, agentsdk.WithFixedToken(authToken))

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			_, err := agentClient.GitSSHKey(ctx)

			if tt.expectError {
				require.Error(t, err)
				var sdkErr *nicloudsdk.Error
				require.ErrorAs(t, err, &sdkErr)
				require.Equal(t, http.StatusForbidden, sdkErr.StatusCode())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
