package nicloud_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestCheckACLPermissions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	t.Cleanup(cancel)

	adminClient, adminUser := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		},
	})
	// Create member and org adminClient
	memberClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, adminUser.OrganizationID)
	memberUser, err := memberClient.User(ctx, nicloudsdk.Me)
	require.NoError(t, err)
	orgAdminClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, adminUser.OrganizationID, rbac.ScopedRoleOrgAdmin(adminUser.OrganizationID))
	orgAdminUser, err := orgAdminClient.User(ctx, nicloudsdk.Me)
	require.NoError(t, err)

	version := nicloudtest.CreateTemplateVersion(t, adminClient, adminUser.OrganizationID, nil)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, adminClient, version.ID)
	template := nicloudtest.CreateTemplate(t, adminClient, adminUser.OrganizationID, version.ID)

	err = adminClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
		UserPerms: map[string]nicloudsdk.TemplateRole{
			memberUser.ID.String(): nicloudsdk.TemplateRoleAdmin,
		},
	})
	require.NoError(t, err)

	const (
		updateSpecificTemplate = "read-specific-template"
	)
	params := map[string]nicloudsdk.AuthorizationCheck{
		updateSpecificTemplate: {
			Object: nicloudsdk.AuthorizationObject{
				ResourceType: nicloudsdk.ResourceTemplate,
				ResourceID:   template.ID.String(),
			},
			Action: nicloudsdk.ActionUpdate,
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
				updateSpecificTemplate: true,
			},
		},
		{
			Name:   "OrgAdmin",
			Client: orgAdminClient,
			UserID: orgAdminUser.ID,
			Check: map[string]bool{
				updateSpecificTemplate: true,
			},
		},
		{
			Name:   "Member",
			Client: memberClient,
			UserID: memberUser.ID,
			Check: map[string]bool{
				updateSpecificTemplate: true,
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
