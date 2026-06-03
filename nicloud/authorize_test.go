package nicloud_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestCheckPermissions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	t.Cleanup(cancel)

	adminClient := nicloudtest.New(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: true,
	})
	// Create adminClient, member, and org adminClient
	adminUser := nicloudtest.CreateFirstUser(t, adminClient)
	memberClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, adminUser.OrganizationID)
	memberUser, err := memberClient.User(ctx, nicloudsdk.Me)
	require.NoError(t, err)
	orgAdminClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, adminUser.OrganizationID, rbac.ScopedRoleOrgAdmin(adminUser.OrganizationID))
	orgAdminUser, err := orgAdminClient.User(ctx, nicloudsdk.Me)
	require.NoError(t, err)

	version := nicloudtest.CreateTemplateVersion(t, adminClient, adminUser.OrganizationID, nil)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, adminClient, version.ID)
	template := nicloudtest.CreateTemplate(t, adminClient, adminUser.OrganizationID, version.ID)

	// With admin, member, and org admin
	const (
		readAllUsers           = "read-all-users"
		readOrgWorkspaces      = "read-org-workspaces"
		readMyself             = "read-myself"
		readOwnWorkspaces      = "read-own-workspaces"
		updateSpecificTemplate = "update-specific-template"
	)
	params := map[string]nicloudsdk.AuthorizationCheck{
		readAllUsers: {
			Object: nicloudsdk.AuthorizationObject{
				ResourceType: nicloudsdk.ResourceUser,
			},
			Action: "read",
		},
		readOrgWorkspaces: {
			Object: nicloudsdk.AuthorizationObject{
				ResourceType:   nicloudsdk.ResourceWorkspace,
				OrganizationID: adminUser.OrganizationID.String(),
			},
			Action: "read",
		},
		readMyself: {
			Object: nicloudsdk.AuthorizationObject{
				ResourceType: nicloudsdk.ResourceUser,
				OwnerID:      "me",
			},
			Action: "read",
		},
		readOwnWorkspaces: {
			Object: nicloudsdk.AuthorizationObject{
				ResourceType:   nicloudsdk.ResourceWorkspace,
				OrganizationID: adminUser.OrganizationID.String(),
				OwnerID:        "me",
			},
			Action: "read",
		},
		updateSpecificTemplate: {
			Object: nicloudsdk.AuthorizationObject{
				ResourceType: nicloudsdk.ResourceTemplate,
				ResourceID:   template.ID.String(),
			},
			Action: "update",
		},
	}

	testCases := []struct {
		Name   string
		Client *nicloudsdk.Client
		UserID uuid.UUID
		Check  nicloudsdk.AuthorizationResponse
	}{
		{
			Name:   "Admin",
			Client: adminClient,
			UserID: adminUser.UserID,
			Check: map[string]bool{
				readAllUsers:           true,
				readOrgWorkspaces:      true,
				readMyself:             true,
				readOwnWorkspaces:      true,
				updateSpecificTemplate: true,
			},
		},
		{
			Name:   "OrgAdmin",
			Client: orgAdminClient,
			UserID: orgAdminUser.ID,
			Check: map[string]bool{
				readAllUsers:           true,
				readOrgWorkspaces:      true,
				readMyself:             true,
				readOwnWorkspaces:      true,
				updateSpecificTemplate: true,
			},
		},
		{
			Name:   "Member",
			Client: memberClient,
			UserID: memberUser.ID,
			Check: map[string]bool{
				readAllUsers:           false,
				readOrgWorkspaces:      false,
				readMyself:             true,
				readOwnWorkspaces:      true,
				updateSpecificTemplate: false,
			},
		},
	}

	for _, c := range testCases {
		t.Run("CheckAuthorization/"+c.Name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			t.Cleanup(cancel)

			resp, err := c.Client.AuthCheck(ctx, nicloudsdk.AuthorizationRequest{Checks: params})
			require.NoError(t, err, "check perms")
			require.Equal(t, c.Check, resp)
		})
	}
}
