package nicloud_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/idpsync"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/runtimeconfig"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/coder/serpent"
)

func TestGetGroupSyncSettings(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, db, user := nicloudenttest.NewWithDatabase(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})
		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		ctx := testutil.Context(t, testutil.WaitShort)
		dbresv := runtimeconfig.OrganizationResolver(user.OrganizationID, runtimeconfig.NewStoreResolver(db))
		entry := runtimeconfig.MustNew[*idpsync.GroupSyncSettings]("group-sync-settings")
		err := entry.SetRuntimeValue(dbauthz.AsSystemRestricted(ctx), dbresv, &idpsync.GroupSyncSettings{Field: "august"})
		require.NoError(t, err)

		settings, err := orgAdmin.GroupIDPSyncSettings(ctx, user.OrganizationID.String())
		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)
	})

	t.Run("Legacy", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.OIDC.GroupField = "legacy-group"
		dv.OIDC.GroupRegexFilter = serpent.Regexp(*regexp.MustCompile("legacy-filter"))
		dv.OIDC.GroupMapping = serpent.Struct[map[string]string]{
			Value: map[string]string{
				"foo": "bar",
			},
		}

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})
		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		ctx := testutil.Context(t, testutil.WaitShort)

		settings, err := orgAdmin.GroupIDPSyncSettings(ctx, user.OrganizationID.String())
		require.NoError(t, err)
		require.Equal(t, dv.OIDC.GroupField.Value(), settings.Field)
		require.Equal(t, dv.OIDC.GroupRegexFilter.String(), settings.RegexFilter.String())
		require.Equal(t, dv.OIDC.GroupMapping.Value, settings.LegacyNameMapping)
	})
}

func TestPatchGroupSyncSettings(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		// Test as org admin
		ctx := testutil.Context(t, testutil.WaitShort)
		settings, err := orgAdmin.PatchGroupIDPSyncSettings(ctx, user.OrganizationID.String(), nicloudsdk.GroupSyncSettings{
			Field: "august",
		})
		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)

		fetchedSettings, err := orgAdmin.GroupIDPSyncSettings(ctx, user.OrganizationID.String())
		require.NoError(t, err)
		require.Equal(t, "august", fetchedSettings.Field)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchGroupIDPSyncSettings(ctx, user.OrganizationID.String(), nicloudsdk.GroupSyncSettings{
			Field: "august",
		})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())

		_, err = member.GroupIDPSyncSettings(ctx, user.OrganizationID.String())
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestPatchGroupSyncConfig(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		orgID := user.OrganizationID
		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, orgID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		mapping := map[string][]uuid.UUID{"wibble": {uuid.New()}}

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := orgAdmin.PatchGroupIDPSyncSettings(ctx, orgID.String(), nicloudsdk.GroupSyncSettings{
			Field:             "wibble",
			RegexFilter:       regexp.MustCompile("wib{2,}le"),
			AutoCreateMissing: false,
			Mapping:           mapping,
		})

		require.NoError(t, err)

		fetchedSettings, err := orgAdmin.GroupIDPSyncSettings(ctx, orgID.String())
		require.NoError(t, err)
		require.Equal(t, "wibble", fetchedSettings.Field)
		require.Equal(t, "wib{2,}le", fetchedSettings.RegexFilter.String())
		require.Equal(t, false, fetchedSettings.AutoCreateMissing)
		require.Equal(t, mapping, fetchedSettings.Mapping)

		ctx = testutil.Context(t, testutil.WaitShort)
		settings, err := orgAdmin.PatchGroupIDPSyncConfig(ctx, orgID.String(), nicloudsdk.PatchGroupIDPSyncConfigRequest{
			Field:             "wobble",
			RegexFilter:       regexp.MustCompile("wob{2,}le"),
			AutoCreateMissing: true,
		})

		require.NoError(t, err)
		require.Equal(t, "wobble", settings.Field)
		require.Equal(t, "wob{2,}le", settings.RegexFilter.String())
		require.Equal(t, true, settings.AutoCreateMissing)
		require.Equal(t, mapping, settings.Mapping)

		fetchedSettings, err = orgAdmin.GroupIDPSyncSettings(ctx, orgID.String())
		require.NoError(t, err)
		require.Equal(t, "wobble", fetchedSettings.Field)
		require.Equal(t, "wob{2,}le", fetchedSettings.RegexFilter.String())
		require.Equal(t, true, fetchedSettings.AutoCreateMissing)
		require.Equal(t, mapping, fetchedSettings.Mapping)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchGroupIDPSyncConfig(ctx, user.OrganizationID.String(), nicloudsdk.PatchGroupIDPSyncConfigRequest{})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestPatchGroupSyncMapping(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		orgID := user.OrganizationID
		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, orgID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))
		// These IDs are easier to visually diff if the test fails than truly random
		// ones.
		orgs := []uuid.UUID{
			uuid.MustParse("00000000-b8bd-46bb-bb6c-6c2b2c0dd2ea"),
			uuid.MustParse("01000000-fbe8-464c-9429-fe01a03f3644"),
			uuid.MustParse("02000000-0926-407b-9998-39af62e3d0c5"),
		}

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := orgAdmin.PatchGroupIDPSyncSettings(ctx, orgID.String(), nicloudsdk.GroupSyncSettings{
			Field:             "wibble",
			RegexFilter:       regexp.MustCompile("wib{2,}le"),
			AutoCreateMissing: true,
			Mapping:           map[string][]uuid.UUID{"wobble": {orgs[0]}},
		})
		require.NoError(t, err)

		ctx = testutil.Context(t, testutil.WaitShort)
		settings, err := orgAdmin.PatchGroupIDPSyncMapping(ctx, orgID.String(), nicloudsdk.PatchGroupIDPSyncMappingRequest{
			Add: []nicloudsdk.IDPSyncMapping[uuid.UUID]{
				{Given: "wibble", Gets: orgs[0]},
				{Given: "wobble", Gets: orgs[1]},
				{Given: "wobble", Gets: orgs[2]},
			},
			// Remove takes priority over Add, so "3" should not actually be added to wooble.
			Remove: []nicloudsdk.IDPSyncMapping[uuid.UUID]{
				{Given: "wobble", Gets: orgs[1]},
			},
		})

		expected := map[string][]uuid.UUID{
			"wibble": {orgs[0]},
			"wobble": {orgs[0], orgs[2]},
		}

		require.NoError(t, err)
		require.Equal(t, expected, settings.Mapping)

		fetchedSettings, err := orgAdmin.GroupIDPSyncSettings(ctx, orgID.String())
		require.NoError(t, err)
		require.Equal(t, "wibble", fetchedSettings.Field)
		require.Equal(t, "wib{2,}le", fetchedSettings.RegexFilter.String())
		require.Equal(t, true, fetchedSettings.AutoCreateMissing)
		require.Equal(t, expected, fetchedSettings.Mapping)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchGroupIDPSyncMapping(ctx, user.OrganizationID.String(), nicloudsdk.PatchGroupIDPSyncMappingRequest{})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestGetRoleSyncSettings(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, _, _, user := nicloudenttest.NewWithAPI(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})
		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		ctx := testutil.Context(t, testutil.WaitShort)
		settings, err := orgAdmin.PatchRoleIDPSyncSettings(ctx, user.OrganizationID.String(), nicloudsdk.RoleSyncSettings{
			Field: "august",
			Mapping: map[string][]string{
				"foo": {"bar"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)
		require.Equal(t, map[string][]string{"foo": {"bar"}}, settings.Mapping)

		settings, err = orgAdmin.RoleIDPSyncSettings(ctx, user.OrganizationID.String())
		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)
		require.Equal(t, map[string][]string{"foo": {"bar"}}, settings.Mapping)
	})
}

func TestPatchRoleSyncSettings(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		// Test as org admin
		ctx := testutil.Context(t, testutil.WaitShort)
		settings, err := orgAdmin.PatchRoleIDPSyncSettings(ctx, user.OrganizationID.String(), nicloudsdk.RoleSyncSettings{
			Field: "august",
		})
		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)

		fetchedSettings, err := orgAdmin.RoleIDPSyncSettings(ctx, user.OrganizationID.String())
		require.NoError(t, err)
		require.Equal(t, "august", fetchedSettings.Field)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchRoleIDPSyncSettings(ctx, user.OrganizationID.String(), nicloudsdk.RoleSyncSettings{
			Field: "august",
		})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())

		_, err = member.RoleIDPSyncSettings(ctx, user.OrganizationID.String())
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestPatchRoleSyncConfig(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		orgID := user.OrganizationID
		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, orgID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		mapping := map[string][]string{"wibble": {"group-01"}}

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := orgAdmin.PatchRoleIDPSyncSettings(ctx, orgID.String(), nicloudsdk.RoleSyncSettings{
			Field:   "wibble",
			Mapping: mapping,
		})

		require.NoError(t, err)

		fetchedSettings, err := orgAdmin.RoleIDPSyncSettings(ctx, orgID.String())
		require.NoError(t, err)
		require.Equal(t, "wibble", fetchedSettings.Field)
		require.Equal(t, mapping, fetchedSettings.Mapping)

		ctx = testutil.Context(t, testutil.WaitShort)
		settings, err := orgAdmin.PatchRoleIDPSyncConfig(ctx, orgID.String(), nicloudsdk.PatchRoleIDPSyncConfigRequest{
			Field: "wobble",
		})

		require.NoError(t, err)
		require.Equal(t, "wobble", settings.Field)
		require.Equal(t, mapping, settings.Mapping)

		fetchedSettings, err = orgAdmin.RoleIDPSyncSettings(ctx, orgID.String())
		require.NoError(t, err)
		require.Equal(t, "wobble", fetchedSettings.Field)
		require.Equal(t, mapping, fetchedSettings.Mapping)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchGroupIDPSyncConfig(ctx, user.OrganizationID.String(), nicloudsdk.PatchGroupIDPSyncConfigRequest{})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestPatchRoleSyncMapping(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		orgID := user.OrganizationID
		orgAdmin, _ := nicloudtest.CreateAnotherUser(t, owner, orgID, rbac.ScopedRoleOrgAdmin(user.OrganizationID))

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := orgAdmin.PatchRoleIDPSyncSettings(ctx, orgID.String(), nicloudsdk.RoleSyncSettings{
			Field:   "wibble",
			Mapping: map[string][]string{"wobble": {"group-00"}},
		})
		require.NoError(t, err)

		ctx = testutil.Context(t, testutil.WaitShort)
		settings, err := orgAdmin.PatchRoleIDPSyncMapping(ctx, orgID.String(), nicloudsdk.PatchRoleIDPSyncMappingRequest{
			Add: []nicloudsdk.IDPSyncMapping[string]{
				{Given: "wibble", Gets: "group-00"},
				{Given: "wobble", Gets: "group-01"},
				{Given: "wobble", Gets: "group-02"},
			},
			// Remove takes priority over Add, so "3" should not actually be added to wooble.
			Remove: []nicloudsdk.IDPSyncMapping[string]{
				{Given: "wobble", Gets: "group-01"},
			},
		})

		expected := map[string][]string{
			"wibble": {"group-00"},
			"wobble": {"group-00", "group-02"},
		}

		require.NoError(t, err)
		require.Equal(t, expected, settings.Mapping)

		fetchedSettings, err := orgAdmin.RoleIDPSyncSettings(ctx, orgID.String())
		require.NoError(t, err)
		require.Equal(t, "wibble", fetchedSettings.Field)
		require.Equal(t, expected, fetchedSettings.Mapping)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchGroupIDPSyncMapping(ctx, user.OrganizationID.String(), nicloudsdk.PatchGroupIDPSyncMappingRequest{})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestGetOrganizationSyncSettings(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, _, _, user := nicloudenttest.NewWithAPI(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		expected := map[string][]uuid.UUID{"foo": {user.OrganizationID}}

		ctx := testutil.Context(t, testutil.WaitShort)
		settings, err := owner.PatchOrganizationIDPSyncSettings(ctx, nicloudsdk.OrganizationSyncSettings{
			Field:   "august",
			Mapping: expected,
		})

		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)
		require.Equal(t, expected, settings.Mapping)

		settings, err = owner.OrganizationIDPSyncSettings(ctx)
		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)
		require.Equal(t, expected, settings.Mapping)
	})
}

func TestPatchOrganizationSyncSettings(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitShort)
		//nolint:gocritic // Only owners can change Organization IdP sync settings
		settings, err := owner.PatchOrganizationIDPSyncSettings(ctx, nicloudsdk.OrganizationSyncSettings{
			Field: "august",
		})
		require.NoError(t, err)
		require.Equal(t, "august", settings.Field)

		fetchedSettings, err := owner.OrganizationIDPSyncSettings(ctx)
		require.NoError(t, err)
		require.Equal(t, "august", fetchedSettings.Field)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchRoleIDPSyncSettings(ctx, user.OrganizationID.String(), nicloudsdk.RoleSyncSettings{
			Field: "august",
		})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())

		_, err = member.RoleIDPSyncSettings(ctx, user.OrganizationID.String())
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestPatchOrganizationSyncConfig(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		mapping := map[string][]uuid.UUID{"wibble": {user.OrganizationID}}

		ctx := testutil.Context(t, testutil.WaitShort)
		//nolint:gocritic // Only owners can change Organization IdP sync settings
		_, err := owner.PatchOrganizationIDPSyncSettings(ctx, nicloudsdk.OrganizationSyncSettings{
			Field:         "wibble",
			AssignDefault: true,
			Mapping:       mapping,
		})

		require.NoError(t, err)

		fetchedSettings, err := owner.OrganizationIDPSyncSettings(ctx)
		require.NoError(t, err)
		require.Equal(t, "wibble", fetchedSettings.Field)
		require.Equal(t, true, fetchedSettings.AssignDefault)
		require.Equal(t, mapping, fetchedSettings.Mapping)

		ctx = testutil.Context(t, testutil.WaitShort)
		settings, err := owner.PatchOrganizationIDPSyncConfig(ctx, nicloudsdk.PatchOrganizationIDPSyncConfigRequest{
			Field: "wobble",
		})

		require.NoError(t, err)
		require.Equal(t, "wobble", settings.Field)
		require.Equal(t, false, settings.AssignDefault)
		require.Equal(t, mapping, settings.Mapping)

		fetchedSettings, err = owner.OrganizationIDPSyncSettings(ctx)
		require.NoError(t, err)
		require.Equal(t, "wobble", fetchedSettings.Field)
		require.Equal(t, false, fetchedSettings.AssignDefault)
		require.Equal(t, mapping, fetchedSettings.Mapping)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchOrganizationIDPSyncConfig(ctx, nicloudsdk.PatchOrganizationIDPSyncConfigRequest{})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}

func TestPatchOrganizationSyncMapping(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		owner, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		// These IDs are easier to visually diff if the test fails than truly random
		// ones.
		orgs := []uuid.UUID{
			uuid.MustParse("00000000-b8bd-46bb-bb6c-6c2b2c0dd2ea"),
			uuid.MustParse("01000000-fbe8-464c-9429-fe01a03f3644"),
			uuid.MustParse("02000000-0926-407b-9998-39af62e3d0c5"),
		}

		ctx := testutil.Context(t, testutil.WaitShort)
		//nolint:gocritic // Only owners can change Organization IdP sync settings
		settings, err := owner.PatchOrganizationIDPSyncMapping(ctx, nicloudsdk.PatchOrganizationIDPSyncMappingRequest{
			Add: []nicloudsdk.IDPSyncMapping[uuid.UUID]{
				{Given: "wibble", Gets: orgs[0]},
				{Given: "wobble", Gets: orgs[0]},
				{Given: "wobble", Gets: orgs[1]},
				{Given: "wobble", Gets: orgs[2]},
			},
			Remove: []nicloudsdk.IDPSyncMapping[uuid.UUID]{
				{Given: "wobble", Gets: orgs[1]},
			},
		})

		expected := map[string][]uuid.UUID{
			"wibble": {orgs[0]},
			"wobble": {orgs[0], orgs[2]},
		}

		require.NoError(t, err)
		require.Equal(t, expected, settings.Mapping)

		fetchedSettings, err := owner.OrganizationIDPSyncSettings(ctx)
		require.NoError(t, err)
		require.Equal(t, expected, fetchedSettings.Mapping)
	})

	t.Run("NotAuthorized", func(t *testing.T) {
		t.Parallel()

		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureCustomRoles:           1,
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		member, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitShort)
		_, err := member.PatchOrganizationIDPSyncMapping(ctx, nicloudsdk.PatchOrganizationIDPSyncMappingRequest{})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusForbidden, apiError.StatusCode())
	})
}
