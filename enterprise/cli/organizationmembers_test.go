package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestRemoveOrganizationMembers(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		ownerClient, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		secondOrganization := nicloudenttest.CreateOrganization(t, ownerClient, nicloudenttest.CreateOrganizationOptions{})
		orgAdminClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, secondOrganization.ID, rbac.ScopedRoleOrgAdmin(secondOrganization.ID))
		_, user := nicloudtest.CreateAnotherUser(t, ownerClient, secondOrganization.ID)

		ctx := testutil.Context(t, testutil.WaitMedium)

		inv, root := clitest.New(t, "organization", "members", "remove", "-O", secondOrganization.ID.String(), user.Username)
		clitest.SetupConfig(t, orgAdminClient, root)

		buf := new(bytes.Buffer)
		inv.Stdout = buf
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		members, err := orgAdminClient.OrganizationMembers(ctx, secondOrganization.ID)
		require.NoError(t, err)

		require.Len(t, members, 2)
	})

	t.Run("UserNotExists", func(t *testing.T) {
		t.Parallel()

		ownerClient := nicloudtest.New(t, &nicloudtest.Options{})
		owner := nicloudtest.CreateFirstUser(t, ownerClient)
		orgAdminClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.ScopedRoleOrgAdmin(owner.OrganizationID))

		ctx := testutil.Context(t, testutil.WaitMedium)

		inv, root := clitest.New(t, "organization", "members", "remove", "-O", owner.OrganizationID.String(), "random_name")
		clitest.SetupConfig(t, orgAdminClient, root)

		buf := new(bytes.Buffer)
		inv.Stdout = buf
		err := inv.WithContext(ctx).Run()
		require.ErrorContains(t, err, "Resource not found or you do not have access to this resource")
	})
}

func TestEnterpriseListOrganizationMembers(t *testing.T) {
	t.Parallel()

	t.Run("CustomRole", func(t *testing.T) {
		t.Parallel()

		ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)
		//nolint:gocritic // only owners can patch roles
		customRole, err := ownerClient.CreateOrganizationRole(ctx, nicloudsdk.Role{
			Name:            "custom",
			OrganizationID:  owner.OrganizationID.String(),
			DisplayName:     "Custom Role",
			SitePermissions: nil,
			OrganizationPermissions: nicloudsdk.CreatePermissions(map[nicloudsdk.RBACResource][]nicloudsdk.RBACAction{
				nicloudsdk.ResourceWorkspace: {nicloudsdk.ActionRead},
			}),
			UserPermissions: nil,
		})
		require.NoError(t, err)

		client, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleUserAdmin(), rbac.RoleIdentifier{
			Name:           customRole.Name,
			OrganizationID: owner.OrganizationID,
		}, rbac.ScopedRoleOrgAdmin(owner.OrganizationID))

		inv, root := clitest.New(t, "organization", "members", "list", "-c", "user id,username,organization roles")
		clitest.SetupConfig(t, client, root)

		buf := new(bytes.Buffer)
		inv.Stdout = buf
		err = inv.WithContext(ctx).Run()
		require.NoError(t, err)
		require.Contains(t, buf.String(), user.Username)
		require.Contains(t, buf.String(), owner.UserID.String())
		// Check the display name is the value in the cli list
		require.Contains(t, buf.String(), customRole.DisplayName)
	})
}

func TestAssignOrganizationMemberRole(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})
		_, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleUserAdmin())

		ctx := testutil.Context(t, testutil.WaitMedium)
		// nolint:gocritic // requires owner role to create
		customRole, err := ownerClient.CreateOrganizationRole(ctx, nicloudsdk.Role{
			Name:            "custom-role",
			OrganizationID:  owner.OrganizationID.String(),
			DisplayName:     "Custom Role",
			SitePermissions: nil,
			OrganizationPermissions: nicloudsdk.CreatePermissions(map[nicloudsdk.RBACResource][]nicloudsdk.RBACAction{
				nicloudsdk.ResourceWorkspace: {nicloudsdk.ActionRead},
			}),
			UserPermissions: nil,
		})
		require.NoError(t, err)

		inv, root := clitest.New(t, "organization", "members", "edit-roles", user.Username, nicloudsdk.RoleOrganizationAdmin, customRole.Name)
		// nolint:gocritic // you cannot change your own roles
		clitest.SetupConfig(t, ownerClient, root)

		buf := new(bytes.Buffer)
		inv.Stdout = buf
		err = inv.WithContext(ctx).Run()
		require.NoError(t, err)
		require.Contains(t, buf.String(), must(rbac.RoleByName(rbac.ScopedRoleOrgAdmin(owner.OrganizationID))).DisplayName)
		require.Contains(t, buf.String(), customRole.DisplayName)
	})
}

func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
