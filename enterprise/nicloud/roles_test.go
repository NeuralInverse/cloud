package nicloud_test

import (
	"bytes"
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/db2sdk"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/slice"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestCustomOrganizationRole(t *testing.T) {
	t.Parallel()
	templateAdminCustom := func(orgID uuid.UUID) nicloudsdk.Role {
		return nicloudsdk.Role{
			Name:           "test-role",
			DisplayName:    "Testing Purposes",
			OrganizationID: orgID.String(),
			// Basically creating a template admin manually
			SitePermissions: nil,
			OrganizationPermissions: nicloudsdk.CreatePermissions(map[nicloudsdk.RBACResource][]nicloudsdk.RBACAction{
				nicloudsdk.ResourceTemplate:  {nicloudsdk.ActionCreate, nicloudsdk.ActionRead, nicloudsdk.ActionUpdate, nicloudsdk.ActionViewInsights},
				nicloudsdk.ResourceFile:      {nicloudsdk.ActionCreate, nicloudsdk.ActionRead},
				nicloudsdk.ResourceWorkspace: {nicloudsdk.ActionRead},
			}),
			UserPermissions: nil,
		}
	}

	// Create, assign, and use a custom role
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		//nolint:gocritic // owner is required for this
		role, err := owner.CreateOrganizationRole(ctx, templateAdminCustom(first.OrganizationID))
		require.NoError(t, err, "upsert role")

		// Assign the custom template admin role
		tmplAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.RoleIdentifier{Name: role.Name, OrganizationID: first.OrganizationID})

		// Assert the role exists
		// TODO: At present user roles are not returned by the user endpoints.
		// 	Changing this might mess up the UI in how it renders the roles on the
		//	users page. When the users endpoint is updated, this should be uncommented.
		// roleNamesF := func(role nicloudsdk.SlimRole) string { return role.Name }
		// require.Contains(t, slice.List(user.Roles, roleNamesF), role.Name)

		// Try to create a template version
		nicloudtest.CreateTemplateVersion(t, tmplAdmin, first.OrganizationID, nil)

		// Verify the role exists in the list
		allRoles, err := tmplAdmin.ListOrganizationRoles(ctx, first.OrganizationID)
		require.NoError(t, err)

		var foundRole nicloudsdk.AssignableRoles
		require.True(t, slices.ContainsFunc(allRoles, func(selected nicloudsdk.AssignableRoles) bool {
			if selected.Name == role.Name {
				foundRole = selected
				return true
			}
			return false
		}), "role missing from org role list")

		require.Len(t, foundRole.SitePermissions, 0)
		require.Len(t, foundRole.OrganizationPermissions, 7)
		require.Len(t, foundRole.UserPermissions, 0)
	})

	// Revoked licenses cannot modify/create custom roles, but they can
	// use the existing roles.
	t.Run("RevokedLicense", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		//nolint:gocritic // owner is required for this
		role, err := owner.CreateOrganizationRole(ctx, templateAdminCustom(first.OrganizationID))
		require.NoError(t, err, "upsert role")

		// Remove the license to block premium functionality
		licenses, err := owner.Licenses(ctx)
		require.NoError(t, err, "get licenses")
		for _, license := range licenses {
			// Should be only 1...
			err := owner.DeleteLicense(ctx, license.ID)
			require.NoError(t, err, "delete license")
		}

		// Verify functionality is lost
		_, err = owner.UpdateOrganizationRole(ctx, templateAdminCustom(first.OrganizationID))
		require.ErrorContains(t, err, "Custom Roles is a Premium feature")

		// Assign the custom template admin role
		tmplAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.RoleIdentifier{Name: role.Name, OrganizationID: first.OrganizationID})

		// Try to create a template version, eg using the custom role
		nicloudtest.CreateTemplateVersion(t, tmplAdmin, first.OrganizationID, nil)
	})

	// Role patches are complete, as in the request overrides the existing role.
	t.Run("RoleOverrides", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)
		//nolint:gocritic // owner is required for this
		role, err := owner.CreateOrganizationRole(ctx, templateAdminCustom(first.OrganizationID))
		require.NoError(t, err, "upsert role")

		// Assign the custom template admin role
		tmplAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.RoleIdentifier{Name: role.Name, OrganizationID: first.OrganizationID})

		// Try to create a template version, eg using the custom role
		nicloudtest.CreateTemplateVersion(t, tmplAdmin, first.OrganizationID, nil)

		//nolint:gocritic // owner is required for this
		newRole := templateAdminCustom(first.OrganizationID)
		// These are all left nil, which sets the custom role to have 0
		// permissions. Omitting this does not "inherit" what already
		// exists.
		newRole.SitePermissions = nil
		newRole.OrganizationPermissions = nil
		newRole.UserPermissions = nil
		_, err = owner.UpdateOrganizationRole(ctx, newRole)
		require.NoError(t, err, "upsert role with override")

		// The role should no longer have template perms
		data, err := echo.TarWithOptions(ctx, tmplAdmin.Logger(), nil)
		require.NoError(t, err)
		file, err := tmplAdmin.Upload(ctx, nicloudsdk.ContentTypeTar, bytes.NewReader(data))
		require.NoError(t, err)
		_, err = tmplAdmin.CreateTemplateVersion(ctx, first.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			FileID:        file.ID,
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.ErrorContains(t, err, "forbidden")
	})

	t.Run("InvalidName", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		//nolint:gocritic // owner is required for this
		_, err := owner.CreateOrganizationRole(ctx, nicloudsdk.Role{
			Name:                    "Bad_Name", // No underscores allowed
			DisplayName:             "Testing Purposes",
			OrganizationID:          first.OrganizationID.String(),
			SitePermissions:         nil,
			OrganizationPermissions: nil,
			UserPermissions:         nil,
		})
		require.ErrorContains(t, err, "Validation")
	})

	t.Run("ReservedName", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		//nolint:gocritic // owner is required for this
		_, err := owner.CreateOrganizationRole(ctx, nicloudsdk.Role{
			Name:                    "owner", // Reserved
			DisplayName:             "Testing Purposes",
			OrganizationID:          first.OrganizationID.String(),
			SitePermissions:         nil,
			OrganizationPermissions: nil,
			UserPermissions:         nil,
		})
		require.ErrorContains(t, err, "Reserved")
	})

	// Attempt to add site & user permissions, which is not allowed
	t.Run("ExcessPermissions", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		siteRole := templateAdminCustom(first.OrganizationID)
		siteRole.SitePermissions = []nicloudsdk.Permission{
			{
				ResourceType: nicloudsdk.ResourceWorkspace,
				Action:       nicloudsdk.ActionRead,
			},
		}

		//nolint:gocritic // owner is required for this
		_, err := owner.CreateOrganizationRole(ctx, siteRole)
		require.ErrorContains(t, err, "site wide permissions")

		userRole := templateAdminCustom(first.OrganizationID)
		userRole.UserPermissions = []nicloudsdk.Permission{
			{
				ResourceType: nicloudsdk.ResourceWorkspace,
				Action:       nicloudsdk.ActionRead,
			},
		}

		//nolint:gocritic // owner is required for this
		_, err = owner.UpdateOrganizationRole(ctx, userRole)
		require.ErrorContains(t, err, "not allowed to assign user permissions")
	})

	// Attempt to add org member permissions, which is not allowed.
	t.Run("MemberPermissions", func(t *testing.T) {
		t.Parallel()

		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		role := templateAdminCustom(first.OrganizationID)
		role.Name = "test-role-member-perms"
		role.OrganizationMemberPermissions = []nicloudsdk.Permission{
			{
				ResourceType: nicloudsdk.ResourceWorkspace,
				Action:       nicloudsdk.ActionRead,
			},
		}

		//nolint:gocritic // we want unrestricted permissions for the test
		_, err := owner.CreateOrganizationRole(ctx, role)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
		require.ErrorContains(t, err, "not allowed to assign organization member permissions for an organization role")
	})

	// System roles are stored in the DB but excluded from the custom
	// roles API, so attempting to delete one returns 404.
	t.Run("DeleteSystemRole", func(t *testing.T) {
		t.Parallel()

		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		//nolint:gocritic // we want unrestricted permissions for the test
		err := owner.DeleteOrganizationRole(ctx, first.OrganizationID, rbac.RoleOrgMember())
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)

		newRole := templateAdminCustom(first.OrganizationID)
		newRole.OrganizationID = "0000" // This is not a valid uuid

		//nolint:gocritic // owner is required for this
		_, err := owner.CreateOrganizationRole(ctx, newRole)
		require.ErrorContains(t, err, "Resource not found")
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		orgAdmin, orgAdminUser := nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.ScopedRoleOrgAdmin(first.OrganizationID))
		ctx := testutil.Context(t, testutil.WaitMedium)

		createdRole, err := orgAdmin.CreateOrganizationRole(ctx, templateAdminCustom(first.OrganizationID))
		require.NoError(t, err, "upsert role")

		//nolint:gocritic // org_admin cannot assign to themselves
		_, err = owner.UpdateOrganizationMemberRoles(ctx, first.OrganizationID, orgAdminUser.ID.String(), nicloudsdk.UpdateRoles{
			// Give the user this custom role, to ensure when it is deleted, the user
			// is ok to be used.
			Roles: []string{createdRole.Name, rbac.ScopedRoleOrgAdmin(first.OrganizationID).Name},
		})
		require.NoError(t, err, "assign custom role to user")

		existingRoles, err := orgAdmin.ListOrganizationRoles(ctx, first.OrganizationID)
		require.NoError(t, err)

		exists := slices.ContainsFunc(existingRoles, func(role nicloudsdk.AssignableRoles) bool {
			return role.Name == createdRole.Name
		})
		require.True(t, exists, "custom role should exist")

		// Delete the role
		err = orgAdmin.DeleteOrganizationRole(ctx, first.OrganizationID, createdRole.Name)
		require.NoError(t, err)

		existingRoles, err = orgAdmin.ListOrganizationRoles(ctx, first.OrganizationID)
		require.NoError(t, err)

		exists = slices.ContainsFunc(existingRoles, func(role nicloudsdk.AssignableRoles) bool {
			return role.Name == createdRole.Name
		})
		require.False(t, exists, "custom role should be deleted")

		// Verify you can still assign roles.
		// There used to be a bug that if a member had a delete role, they
		// could not be assigned roles anymore.
		//nolint:gocritic // org_admin cannot assign to themselves
		_, err = owner.UpdateOrganizationMemberRoles(ctx, first.OrganizationID, orgAdminUser.ID.String(), nicloudsdk.UpdateRoles{
			Roles: []string{rbac.ScopedRoleOrgAdmin(first.OrganizationID).Name},
		})
		require.NoError(t, err)
	})

	// Verify deleting a custom role cascades to all members
	t.Run("DeleteRoleCascadeMembers", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles: 1,
				},
			},
		})

		orgAdmin, orgAdminUser := nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.ScopedRoleOrgAdmin(first.OrganizationID))
		ctx := testutil.Context(t, testutil.WaitMedium)

		createdRole, err := orgAdmin.CreateOrganizationRole(ctx, templateAdminCustom(first.OrganizationID))
		require.NoError(t, err, "upsert role")

		customRoleIdentifier := rbac.RoleIdentifier{
			Name:           createdRole.Name,
			OrganizationID: first.OrganizationID,
		}

		// Create a few members with the role
		nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, customRoleIdentifier)
		nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.ScopedRoleOrgAdmin(first.OrganizationID), customRoleIdentifier)
		nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.ScopedRoleOrgTemplateAdmin(first.OrganizationID), rbac.ScopedRoleOrgAuditor(first.OrganizationID), customRoleIdentifier)

		// Verify members have the custom role
		originalMembers, err := orgAdmin.OrganizationMembers(ctx, first.OrganizationID)
		require.NoError(t, err)
		require.Len(t, originalMembers, 5) // 3 members + org admin + owner
		for _, member := range originalMembers {
			if member.UserID == orgAdminUser.ID || member.UserID == first.UserID {
				continue
			}

			require.True(t, slices.ContainsFunc(member.Roles, func(role nicloudsdk.SlimRole) bool {
				return role.Name == customRoleIdentifier.Name
			}), "member should have custom role")
		}

		err = orgAdmin.DeleteOrganizationRole(ctx, first.OrganizationID, createdRole.Name)
		require.NoError(t, err)

		// Verify the role was removed from all members
		members, err := orgAdmin.OrganizationMembers(ctx, first.OrganizationID)
		require.NoError(t, err)
		require.Len(t, members, 5) // 3 members + org admin + owner
		for _, member := range members {
			require.False(t, slices.ContainsFunc(member.Roles, func(role nicloudsdk.SlimRole) bool {
				return role.Name == customRoleIdentifier.Name
			}), "role should be removed from all users")

			// Verify the rest of the member's roles are unchanged
			original := originalMembers[slices.IndexFunc(originalMembers, func(haystack nicloudsdk.OrganizationMemberWithUserData) bool {
				return haystack.UserID == member.UserID
			})]
			originalWithoutCustom := slices.DeleteFunc(original.Roles, func(role nicloudsdk.SlimRole) bool {
				return role.Name == customRoleIdentifier.Name
			})
			require.ElementsMatch(t, originalWithoutCustom, member.Roles, "original roles are unchanged")
		}
	})
}

func TestListRoles(t *testing.T) {
	t.Parallel()

	dv := nicloudtest.DeploymentValues(t)

	client, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			DeploymentValues: dv,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureExternalProvisionerDaemons: 1,
				nicloudsdk.FeatureMultipleOrganizations:      1,
			},
		},
	})

	// Create owner, member, and org admin
	member, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
	orgAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.ScopedRoleOrgAdmin(owner.OrganizationID))

	otherOrg := nicloudenttest.CreateOrganization(t, client, nicloudenttest.CreateOrganizationOptions{})

	const notFound = "Resource not found"
	testCases := []struct {
		Name            string
		Client          *nicloudsdk.Client
		APICall         func(context.Context) ([]nicloudsdk.AssignableRoles, error)
		ExpectedRoles   []nicloudsdk.AssignableRoles
		AuthorizedError string
	}{
		{
			// Members cannot assign any roles
			Name: "MemberListSite",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				x, err := member.ListSiteRoles(ctx)
				return x, err
			},
			ExpectedRoles: convertRoles(map[rbac.RoleIdentifier]bool{
				{Name: nicloudsdk.RoleOwner}:         false,
				{Name: nicloudsdk.RoleAuditor}:       false,
				{Name: nicloudsdk.RoleTemplateAdmin}: false,
				{Name: nicloudsdk.RoleUserAdmin}:     false,
			}),
		},
		{
			Name: "OrgMemberListOrg",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				return member.ListOrganizationRoles(ctx, owner.OrganizationID)
			},
			ExpectedRoles: convertRoles(map[rbac.RoleIdentifier]bool{
				{Name: nicloudsdk.RoleOrganizationAdmin, OrganizationID: owner.OrganizationID}:                false,
				{Name: nicloudsdk.RoleOrganizationAuditor, OrganizationID: owner.OrganizationID}:              false,
				{Name: nicloudsdk.RoleOrganizationTemplateAdmin, OrganizationID: owner.OrganizationID}:        false,
				{Name: nicloudsdk.RoleOrganizationUserAdmin, OrganizationID: owner.OrganizationID}:            false,
				{Name: nicloudsdk.RoleOrganizationWorkspaceCreationBan, OrganizationID: owner.OrganizationID}: false,
				{Name: nicloudsdk.RoleAgentsAccess, OrganizationID: owner.OrganizationID}:                     false,
			}),
		},
		{
			Name: "NonOrgMemberListOrg",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				return member.ListOrganizationRoles(ctx, otherOrg.ID)
			},
			AuthorizedError: notFound,
		},
		// Org admin
		{
			Name: "OrgAdminListSite",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				return orgAdmin.ListSiteRoles(ctx)
			},
			ExpectedRoles: convertRoles(map[rbac.RoleIdentifier]bool{
				{Name: nicloudsdk.RoleOwner}:         false,
				{Name: nicloudsdk.RoleAuditor}:       false,
				{Name: nicloudsdk.RoleTemplateAdmin}: false,
				{Name: nicloudsdk.RoleUserAdmin}:     false,
			}),
		},
		{
			Name: "OrgAdminListOrg",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				return orgAdmin.ListOrganizationRoles(ctx, owner.OrganizationID)
			},
			ExpectedRoles: convertRoles(map[rbac.RoleIdentifier]bool{
				{Name: nicloudsdk.RoleOrganizationAdmin, OrganizationID: owner.OrganizationID}:                true,
				{Name: nicloudsdk.RoleOrganizationAuditor, OrganizationID: owner.OrganizationID}:              true,
				{Name: nicloudsdk.RoleOrganizationTemplateAdmin, OrganizationID: owner.OrganizationID}:        true,
				{Name: nicloudsdk.RoleOrganizationUserAdmin, OrganizationID: owner.OrganizationID}:            true,
				{Name: nicloudsdk.RoleOrganizationWorkspaceCreationBan, OrganizationID: owner.OrganizationID}: true,
				{Name: nicloudsdk.RoleAgentsAccess, OrganizationID: owner.OrganizationID}:                     true,
			}),
		},
		{
			Name: "OrgAdminListOtherOrg",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				return orgAdmin.ListOrganizationRoles(ctx, otherOrg.ID)
			},
			AuthorizedError: notFound,
		},
		// Admin
		{
			Name: "AdminListSite",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				return client.ListSiteRoles(ctx)
			},
			ExpectedRoles: convertRoles(map[rbac.RoleIdentifier]bool{
				{Name: nicloudsdk.RoleOwner}:         true,
				{Name: nicloudsdk.RoleAuditor}:       true,
				{Name: nicloudsdk.RoleTemplateAdmin}: true,
				{Name: nicloudsdk.RoleUserAdmin}:     true,
			}),
		},
		{
			Name: "AdminListOrg",
			APICall: func(ctx context.Context) ([]nicloudsdk.AssignableRoles, error) {
				return client.ListOrganizationRoles(ctx, owner.OrganizationID)
			},
			ExpectedRoles: convertRoles(map[rbac.RoleIdentifier]bool{
				{Name: nicloudsdk.RoleOrganizationAdmin, OrganizationID: owner.OrganizationID}:                true,
				{Name: nicloudsdk.RoleOrganizationAuditor, OrganizationID: owner.OrganizationID}:              true,
				{Name: nicloudsdk.RoleOrganizationTemplateAdmin, OrganizationID: owner.OrganizationID}:        true,
				{Name: nicloudsdk.RoleOrganizationUserAdmin, OrganizationID: owner.OrganizationID}:            true,
				{Name: nicloudsdk.RoleOrganizationWorkspaceCreationBan, OrganizationID: owner.OrganizationID}: true,
				{Name: nicloudsdk.RoleAgentsAccess, OrganizationID: owner.OrganizationID}:                     true,
			}),
		},
	}

	for _, c := range testCases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			roles, err := c.APICall(ctx)
			if c.AuthorizedError != "" {
				var apiErr *nicloudsdk.Error
				require.ErrorAs(t, err, &apiErr)
				require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
				require.Contains(t, apiErr.Message, c.AuthorizedError)
			} else {
				require.NoError(t, err)
				ignorePerms := func(f nicloudsdk.AssignableRoles) nicloudsdk.AssignableRoles {
					return nicloudsdk.AssignableRoles{
						Role: nicloudsdk.Role{
							Name:        f.Name,
							DisplayName: f.DisplayName,
						},
						Assignable: f.Assignable,
						BuiltIn:    true,
					}
				}
				expected := slice.List(c.ExpectedRoles, ignorePerms)
				found := slice.List(roles, ignorePerms)
				require.ElementsMatch(t, expected, found)
			}
		})
	}
}

func convertRole(roleName rbac.RoleIdentifier) nicloudsdk.Role {
	role, _ := rbac.RoleByName(roleName)
	return db2sdk.RBACRole(role)
}

func convertRoles(assignableRoles map[rbac.RoleIdentifier]bool) []nicloudsdk.AssignableRoles {
	converted := make([]nicloudsdk.AssignableRoles, 0, len(assignableRoles))
	for roleName, assignable := range assignableRoles {
		role := convertRole(roleName)
		converted = append(converted, nicloudsdk.AssignableRoles{
			Role:       role,
			Assignable: assignable,
		})
	}
	return converted
}
