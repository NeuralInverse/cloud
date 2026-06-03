package acl_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbgen"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/acl"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestOK(t *testing.T) {
	t.Parallel()

	db, _ := dbtestutil.NewDB(t)
	o := dbgen.Organization(t, db, database.Organization{})
	g := dbgen.Group(t, db, database.Group{OrganizationID: o.ID})
	u := dbgen.User(t, db, database.User{})
	ctx := testutil.Context(t, testutil.WaitShort)

	update := nicloudsdk.UpdateWorkspaceACL{
		UserRoles: map[string]nicloudsdk.WorkspaceRole{
			u.ID.String(): nicloudsdk.WorkspaceRoleAdmin,
			// An unknown ID is allowed if and only if the specified role is either
			// nicloudsdk.WorkspaceRoleDeleted or nicloudsdk.TemplateRoleDeleted.
			uuid.NewString(): nicloudsdk.WorkspaceRoleDeleted,
		},
		GroupRoles: map[string]nicloudsdk.WorkspaceRole{
			g.ID.String(): nicloudsdk.WorkspaceRoleAdmin,
			// An unknown ID is allowed if and only if the specified role is either
			// nicloudsdk.WorkspaceRoleDeleted or nicloudsdk.TemplateRoleDeleted.
			uuid.NewString(): nicloudsdk.WorkspaceRoleDeleted,
		},
	}
	errors := acl.Validate(ctx, db, nicloud.WorkspaceACLUpdateValidator(update))
	require.Empty(t, errors)
}

func TestDeniesUnknownIDs(t *testing.T) {
	t.Parallel()

	db, _ := dbtestutil.NewDB(t)
	ctx := testutil.Context(t, testutil.WaitShort)

	update := nicloudsdk.UpdateWorkspaceACL{
		UserRoles: map[string]nicloudsdk.WorkspaceRole{
			uuid.NewString(): nicloudsdk.WorkspaceRoleAdmin,
		},
		GroupRoles: map[string]nicloudsdk.WorkspaceRole{
			uuid.NewString(): nicloudsdk.WorkspaceRoleAdmin,
		},
	}
	errors := acl.Validate(ctx, db, nicloud.WorkspaceACLUpdateValidator(update))
	require.Len(t, errors, 2)
	require.Equal(t, errors[0].Field, "group_roles")
	require.ErrorContains(t, errors[0], "does not exist")
	require.Equal(t, errors[1].Field, "user_roles")
	require.ErrorContains(t, errors[1], "does not exist")
}

func TestDeniesUnknownRolesAndInvalidIDs(t *testing.T) {
	t.Parallel()

	db, _ := dbtestutil.NewDB(t)
	ctx := testutil.Context(t, testutil.WaitShort)

	update := nicloudsdk.UpdateWorkspaceACL{
		UserRoles: map[string]nicloudsdk.WorkspaceRole{
			"Quifrey": "level 5",
		},
		GroupRoles: map[string]nicloudsdk.WorkspaceRole{
			"apprentices": "level 2",
		},
	}
	errors := acl.Validate(ctx, db, nicloud.WorkspaceACLUpdateValidator(update))
	require.Len(t, errors, 4)
	require.Equal(t, errors[0].Field, "group_roles")
	require.ErrorContains(t, errors[0], "role \"level 2\" is not a valid workspace role")
	require.Equal(t, errors[1].Field, "group_roles")
	require.ErrorContains(t, errors[1], "not a valid UUID")
	require.Equal(t, errors[2].Field, "user_roles")
	require.ErrorContains(t, errors[2], "role \"level 5\" is not a valid workspace role")
	require.Equal(t, errors[3].Field, "user_roles")
	require.ErrorContains(t, errors[3], "not a valid UUID")
}
