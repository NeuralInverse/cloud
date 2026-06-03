package nicloud_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/agentsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

// TestAgentGitSSHKeyCustomRoles tests that the agent can fetch its git ssh key when
// the user has a custom role in a second workspace.
func TestAgentGitSSHKeyCustomRoles(t *testing.T) {
	t.Parallel()

	owner, _ := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureCustomRoles:                1,
				nicloudsdk.FeatureMultipleOrganizations:      1,
				nicloudsdk.FeatureExternalProvisionerDaemons: 1,
			},
		},
	})

	// When custom roles exist in a second organization
	org := nicloudenttest.CreateOrganization(t, owner, nicloudenttest.CreateOrganizationOptions{
		IncludeProvisionerDaemon: true,
	})

	ctx := testutil.Context(t, testutil.WaitShort)
	//nolint:gocritic // required to make orgs
	newRole, err := owner.CreateOrganizationRole(ctx, nicloudsdk.Role{
		Name:            "custom",
		OrganizationID:  org.ID.String(),
		DisplayName:     "",
		SitePermissions: nil,
		OrganizationPermissions: nicloudsdk.CreatePermissions(map[nicloudsdk.RBACResource][]nicloudsdk.RBACAction{
			nicloudsdk.ResourceTemplate: {nicloudsdk.ActionRead, nicloudsdk.ActionCreate, nicloudsdk.ActionUpdate},
		}),
		UserPermissions: nil,
	})
	require.NoError(t, err)

	// Create the new user
	client, _ := nicloudtest.CreateAnotherUser(t, owner, org.ID, rbac.RoleIdentifier{Name: newRole.Name, OrganizationID: org.ID})

	// Create the workspace + agent
	authToken := uuid.NewString()
	version := nicloudtest.CreateTemplateVersion(t, client, org.ID, &echo.Responses{
		Parse:          echo.ParseComplete,
		ProvisionPlan:  echo.PlanComplete,
		ProvisionGraph: echo.ProvisionGraphWithAgent(authToken),
	})
	project := nicloudtest.CreateTemplate(t, client, org.ID, version.ID)
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
