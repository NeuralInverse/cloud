package nicloud_test

import (
	"context"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud"
	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/rolestore"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/slice"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestCreateGroup(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name:      "hi",
			AvatarURL: "https://example.com",
		})
		require.NoError(t, err)
		require.Equal(t, "hi", group.Name)
		require.Equal(t, "https://example.com", group.AvatarURL)
		require.Empty(t, group.Members)
		require.Empty(t, group.DisplayName)
		require.NotEqual(t, uuid.Nil.String(), group.ID.String())
	})

	t.Run("Audit", func(t *testing.T) {
		t.Parallel()

		auditor := audit.NewMock()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			AuditLogging: true,
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				Auditor:                  auditor,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
					nicloudsdk.FeatureAuditLog:     1,
				},
			},
		})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

		ctx := testutil.Context(t, testutil.WaitLong)

		numLogs := len(auditor.AuditLogs())
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)
		numLogs++
		require.Len(t, auditor.AuditLogs(), numLogs)
		require.True(t, auditor.Contains(t, database.AuditLog{
			Action:     database.AuditActionCreate,
			ResourceID: group.ID,
		}))
	})

	t.Run("Conflict", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		_, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		_, err = userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusConflict, cerr.StatusCode())
	})

	t.Run("ReservedName", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		_, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "new",
		})

		require.Error(t, err)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})

	t.Run("allUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		_, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: database.EveryoneGroup,
		})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})
}

func TestPatchGroup(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		const displayName = "foobar"
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name:           "ff7dcee2-e7c4-4bc4-a9e4-84870770e4c5", // GUID should fit.
			AvatarURL:      "https://example.com",
			QuotaAllowance: 10,
			DisplayName:    "",
		})
		require.NoError(t, err)
		require.Equal(t, 10, group.QuotaAllowance)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			Name:           "ddd502d2-2984-4724-b5bf-1109a4d7462d", // GUID should fit.
			AvatarURL:      ptr.Ref("https://google.com"),
			QuotaAllowance: ptr.Ref(20),
			DisplayName:    ptr.Ref(displayName),
		})
		require.NoError(t, err)
		require.Equal(t, displayName, group.DisplayName)
		require.Equal(t, "ddd502d2-2984-4724-b5bf-1109a4d7462d", group.Name)
		require.Equal(t, "https://google.com", group.AvatarURL)
		require.Equal(t, 20, group.QuotaAllowance)
	})

	t.Run("DisplayNameUnchanged", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		const displayName = "foobar"
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name:           "hi",
			AvatarURL:      "https://example.com",
			QuotaAllowance: 10,
			DisplayName:    displayName,
		})
		require.NoError(t, err)
		require.Equal(t, 10, group.QuotaAllowance)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			Name:           "bye",
			AvatarURL:      ptr.Ref("https://google.com"),
			QuotaAllowance: ptr.Ref(20),
		})
		require.NoError(t, err)
		require.Equal(t, displayName, group.DisplayName)
		require.Equal(t, "bye", group.Name)
		require.Equal(t, "https://google.com", group.AvatarURL)
		require.Equal(t, 20, group.QuotaAllowance)
	})

	// The FE sends a request from the edit page where the old name == new name.
	// This should pass since it's not really an error to update a group name
	// to itself.
	t.Run("SameNameOK", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)
		require.Equal(t, "hi", group.Name)
	})

	t.Run("AddUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user2.ID.String(), user3.ID.String()},
		})
		require.NoError(t, err)
		require.Contains(t, group.Members, user2.ReducedUser)
		require.Contains(t, group.Members, user3.ReducedUser)
	})

	t.Run("RemoveUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user4 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user2.ID.String(), user3.ID.String(), user4.ID.String()},
		})
		require.NoError(t, err)
		require.Contains(t, group.Members, user2.ReducedUser)
		require.Contains(t, group.Members, user3.ReducedUser)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			RemoveUsers: []string{user2.ID.String(), user3.ID.String()},
		})
		require.NoError(t, err)
		require.NotContains(t, group.Members, user2.ReducedUser)
		require.NotContains(t, group.Members, user3.ReducedUser)
		require.Contains(t, group.Members, user4.ReducedUser)
	})

	t.Run("Audit", func(t *testing.T) {
		t.Parallel()

		auditor := audit.NewMock()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			AuditLogging: true,
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				Auditor:                  auditor,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
					nicloudsdk.FeatureAuditLog:     1,
				},
			},
		})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)

		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		numLogs := len(auditor.AuditLogs())
		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			Name: "bye",
		})
		require.NoError(t, err)
		numLogs++

		require.Len(t, auditor.AuditLogs(), numLogs)
		require.Equal(t, database.AuditActionWrite, auditor.AuditLogs()[numLogs-1].Action)
		require.Equal(t, group.ID, auditor.AuditLogs()[numLogs-1].ResourceID)
	})
	t.Run("NameConflict", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group1, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name:      "hi",
			AvatarURL: "https://example.com",
		})
		require.NoError(t, err)

		group2, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "bye",
		})
		require.NoError(t, err)

		group1, err = userAdminClient.PatchGroup(ctx, group1.ID, nicloudsdk.PatchGroupRequest{
			Name:      group2.Name,
			AvatarURL: ptr.Ref("https://google.com"),
		})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusConflict, cerr.StatusCode())
	})

	t.Run("UserNotExist", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{uuid.NewString()},
		})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	t.Run("MalformedUUID", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{"yeet"},
		})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	t.Run("AddDuplicateUser", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user2.ID.String(), user2.ID.String()},
		})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)

		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	t.Run("ReservedName", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			Name: database.EveryoneGroup,
		})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	// For quotas to work with prebuilds, it's currently required to add the
	// prebuilds user into a group with a quota allowance.
	// See: docs/admin/templates/extending-templates/prebuilt-workspaces.md
	t.Run("PrebuildsUser", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name:           "prebuilds",
			QuotaAllowance: 123,
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			Name:     "prebuilds",
			AddUsers: []string{database.PrebuildsSystemUserID.String()},
		})
		require.NoError(t, err)
	})

	t.Run("Everyone", func(t *testing.T) {
		t.Parallel()
		t.Run("NoUpdateName", func(t *testing.T) {
			t.Parallel()

			client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
			ctx := testutil.Context(t, testutil.WaitLong)
			_, err := userAdminClient.PatchGroup(ctx, user.OrganizationID, nicloudsdk.PatchGroupRequest{
				Name: "hi",
			})
			require.Error(t, err)
			cerr, ok := nicloudsdk.AsError(err)
			require.True(t, ok)
			require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
		})

		t.Run("NoUpdateDisplayName", func(t *testing.T) {
			t.Parallel()

			client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
			ctx := testutil.Context(t, testutil.WaitLong)
			_, err := userAdminClient.PatchGroup(ctx, user.OrganizationID, nicloudsdk.PatchGroupRequest{
				DisplayName: ptr.Ref("hi"),
			})
			require.Error(t, err)
			cerr, ok := nicloudsdk.AsError(err)
			require.True(t, ok)
			require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
		})

		t.Run("NoAddUsers", func(t *testing.T) {
			t.Parallel()

			client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
			_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

			ctx := testutil.Context(t, testutil.WaitLong)
			_, err := userAdminClient.PatchGroup(ctx, user.OrganizationID, nicloudsdk.PatchGroupRequest{
				AddUsers: []string{user2.ID.String()},
			})
			require.Error(t, err)
			cerr, ok := nicloudsdk.AsError(err)
			require.True(t, ok)
			require.Equal(t, http.StatusForbidden, cerr.StatusCode())
		})

		t.Run("NoRmUsers", func(t *testing.T) {
			t.Parallel()

			client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

			ctx := testutil.Context(t, testutil.WaitLong)
			_, err := userAdminClient.PatchGroup(ctx, user.OrganizationID, nicloudsdk.PatchGroupRequest{
				RemoveUsers: []string{user.UserID.String()},
			})
			require.Error(t, err)
			cerr, ok := nicloudsdk.AsError(err)
			require.True(t, ok)
			require.Equal(t, http.StatusForbidden, cerr.StatusCode())
		})

		t.Run("UpdateQuota", func(t *testing.T) {
			t.Parallel()

			client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

			ctx := testutil.Context(t, testutil.WaitLong)
			group, err := userAdminClient.Group(ctx, user.OrganizationID, nicloudsdk.GroupRequest{})
			require.NoError(t, err)

			require.Equal(t, 0, group.QuotaAllowance)

			expectedQuota := 123
			group, err = userAdminClient.PatchGroup(ctx, user.OrganizationID, nicloudsdk.PatchGroupRequest{
				QuotaAllowance: ptr.Ref(expectedQuota),
			})
			require.NoError(t, err)
			require.Equal(t, expectedQuota, group.QuotaAllowance)
		})
	})
}

func normalizeAllGroups(groups []nicloudsdk.Group) {
	for i := range groups {
		normalizeGroupMembers(&groups[i])
	}
}

// normalizeGroupMembers removes comparison noise from the group members.
func normalizeGroupMembers(group *nicloudsdk.Group) {
	for i := range group.Members {
		group.Members[i].LastSeenAt = time.Time{}
		group.Members[i].CreatedAt = time.Time{}
		group.Members[i].UpdatedAt = time.Time{}
	}
	sort.Slice(group.Members, func(i, j int) bool {
		return group.Members[i].ID.String() < group.Members[j].ID.String()
	})
}

// TODO: test auth.
func TestGroup(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		ggroup, err := userAdminClient.Group(ctx, group.ID, nicloudsdk.GroupRequest{})
		require.NoError(t, err)
		require.Equal(t, group, ggroup)
	})

	t.Run("ByName", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		ggroup, err := userAdminClient.GroupByOrgAndName(ctx, group.OrganizationID, group.Name)
		require.NoError(t, err)
		require.Equal(t, group, ggroup)
	})

	t.Run("WithUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user2.ID.String(), user3.ID.String()},
		})
		require.NoError(t, err)
		require.Contains(t, group.Members, user2.ReducedUser)
		require.Contains(t, group.Members, user3.ReducedUser)

		ggroup, err := userAdminClient.Group(ctx, group.ID, nicloudsdk.GroupRequest{})
		require.NoError(t, err)
		normalizeGroupMembers(&group)
		normalizeGroupMembers(&ggroup)

		require.Equal(t, group, ggroup)
	})

	t.Run("WithoutMembers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user2.ID.String(), user3.ID.String()},
		})
		require.NoError(t, err)
		require.Contains(t, group.Members, user2.ReducedUser)
		require.Contains(t, group.Members, user3.ReducedUser)

		ggroup, err := userAdminClient.Group(ctx, group.ID, nicloudsdk.GroupRequest{
			ExcludeMembers: true,
		})
		require.NoError(t, err)
		require.Len(t, ggroup.Members, 0)
	})

	t.Run("RegularUserReadGroup", func(t *testing.T) {
		t.Parallel()

		t.Run("WorkspaceSharingEnabled", func(t *testing.T) {
			t.Parallel()

			client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureTemplateRBAC: 1,
				},
			}})
			client1, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

			ctx := testutil.Context(t, testutil.WaitLong)
			//nolint:gocritic // test setup
			group, err := client.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
				Name: "hi",
			})
			require.NoError(t, err)

			ggroup, err := client1.Group(ctx, group.ID, nicloudsdk.GroupRequest{})
			require.NoError(t, err, "regular users can read groups unless workspace sharing is disabled")
			normalizeGroupMembers(&group)
			normalizeGroupMembers(&ggroup)
			require.Equal(t, group, ggroup)
		})

		t.Run("WorkspaceSharingDisabled", func(t *testing.T) {
			t.Parallel()

			db, ps, sqlDB := dbtestutil.NewDBWithSQLDB(t)
			client, _, api, user := nicloudenttest.NewWithAPI(t, &nicloudenttest.Options{
				Options: &nicloudtest.Options{
					Database: db,
					Pubsub:   ps,
				},
				LicenseOptions: &nicloudenttest.LicenseOptions{
					Features: license.Features{
						nicloudsdk.FeatureTemplateRBAC: 1,
					},
				},
			})
			ctx := testutil.Context(t, testutil.WaitLong)
			_, err := sqlDB.ExecContext(ctx, "UPDATE organizations SET shareable_workspace_owners = 'none' WHERE id = $1", user.OrganizationID)
			require.NoError(t, err)

			//nolint:gocritic // ReconcileOrgMemberRole needs the system:update
			// permission that the test context doesn't have.
			sysCtx := dbauthz.AsSystemRestricted(ctx)
			_, _, err = rolestore.ReconcileSystemRole(sysCtx, api.Database, database.CustomRole{
				Name: rbac.RoleOrgMember(),
				OrganizationID: uuid.NullUUID{
					UUID:  user.OrganizationID,
					Valid: true,
				},
			}, database.Organization{ShareableWorkspaceOwners: database.ShareableWorkspaceOwnersNone})
			require.NoError(t, err)

			client1, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

			//nolint:gocritic // test setup
			group, err := client.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
				Name: "hi",
			})
			require.NoError(t, err)

			_, err = client1.Group(ctx, group.ID, nicloudsdk.GroupRequest{})
			require.Error(t, err, "regular users cannot read groups when workspace sharing is disabled")
			cerr, ok := nicloudsdk.AsError(err)
			require.True(t, ok)
			require.Equal(t, http.StatusNotFound, cerr.StatusCode())
		})
	})

	t.Run("FilterDeletedUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

		_, user1 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user1.ID.String(), user2.ID.String()},
		})
		require.NoError(t, err)
		require.Contains(t, group.Members, user1.ReducedUser)
		require.Contains(t, group.Members, user2.ReducedUser)

		err = userAdminClient.DeleteUser(ctx, user1.ID)
		require.NoError(t, err)

		group, err = userAdminClient.Group(ctx, group.ID, nicloudsdk.GroupRequest{})
		require.NoError(t, err)
		require.NotContains(t, group.Members, user1.ReducedUser)
	})

	t.Run("IncludeSuspendedAndDormantUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

		_, user1 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitLong)
		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user1.ID.String(), user2.ID.String()},
		})
		require.NoError(t, err)
		require.Len(t, group.Members, 2)
		require.Contains(t, group.Members, user1.ReducedUser)
		require.Contains(t, group.Members, user2.ReducedUser)

		user1, err = userAdminClient.UpdateUserStatus(ctx, user1.ID.String(), nicloudsdk.UserStatusSuspended)
		require.NoError(t, err)

		group, err = userAdminClient.Group(ctx, group.ID, nicloudsdk.GroupRequest{})
		require.NoError(t, err)
		require.Len(t, group.Members, 2)
		require.Contains(t, group.Members, user1.ReducedUser)
		require.Contains(t, group.Members, user2.ReducedUser)

		// cannot explicitly set a dormant user status so must create a new user
		anotherUser, err := userAdminClient.CreateUserWithOrgs(ctx, nicloudsdk.CreateUserRequestWithOrgs{
			Email:           "coder@cloud.neuralinverse.com",
			Username:        "coder",
			Password:        "SomeStrongPassword!",
			OrganizationIDs: []uuid.UUID{user.OrganizationID},
		})
		require.NoError(t, err)

		// Ensure that new user has dormant account
		require.Equal(t, nicloudsdk.UserStatusDormant, anotherUser.Status)

		group, _ = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{anotherUser.ID.String()},
		})

		group, err = userAdminClient.Group(ctx, group.ID, nicloudsdk.GroupRequest{})
		require.NoError(t, err)
		require.Len(t, group.Members, 3)
		require.Contains(t, group.Members, user1.ReducedUser)
		require.Contains(t, group.Members, user2.ReducedUser)
	})

	t.Run("ByIDs", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

		ctx := testutil.Context(t, testutil.WaitLong)
		groupA, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "group-a",
		})
		require.NoError(t, err)

		groupB, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "group-b",
		})
		require.NoError(t, err)

		// group-c should be omitted from the filter
		_, err = userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "group-c",
		})
		require.NoError(t, err)

		found, err := userAdminClient.Groups(ctx, nicloudsdk.GroupArguments{
			GroupIDs: []uuid.UUID{groupA.ID, groupB.ID},
		})
		require.NoError(t, err)

		foundIDs := slice.List(found, func(g nicloudsdk.Group) uuid.UUID {
			return g.ID
		})

		require.ElementsMatch(t, []uuid.UUID{groupA.ID, groupB.ID}, foundIDs)
	})

	t.Run("everyoneGroupReturnsEmpty", func(t *testing.T) {
		t.Parallel()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		_, user1 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		ctx := testutil.Context(t, testutil.WaitLong)

		// nolint:gocritic // "This client is operating as the owner user" is fine in this case.
		prebuildsUser, err := client.User(ctx, database.PrebuildsSystemUserID.String())
		require.NoError(t, err)
		// The 'Everyone' group always has an ID that matches the organization ID.
		group, err := userAdminClient.Group(ctx, user.OrganizationID, nicloudsdk.GroupRequest{})
		require.NoError(t, err)
		require.Len(t, group.Members, 4)
		require.Equal(t, "Everyone", group.Name)
		require.Equal(t, user.OrganizationID, group.OrganizationID)
		require.Contains(t, group.Members, user1.ReducedUser)
		require.Contains(t, group.Members, user2.ReducedUser)
		require.NotContains(t, group.Members, prebuildsUser.ReducedUser)
	})
}

// TODO: test auth.
func TestGroups(t *testing.T) {
	t.Parallel()

	// 5 users
	// 2 custom groups + original org group
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user4 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		user5Client, user5 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		ctx := testutil.Context(t, testutil.WaitLong)
		group1, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		group2, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hey",
		})
		require.NoError(t, err)

		group1, err = userAdminClient.PatchGroup(ctx, group1.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user2.ID.String(), user3.ID.String()},
		})
		require.NoError(t, err)
		normalizeGroupMembers(&group1)

		group2, err = userAdminClient.PatchGroup(ctx, group2.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user4.ID.String(), user5.ID.String()},
		})
		require.NoError(t, err)
		normalizeGroupMembers(&group2)

		// Fetch everyone group for comparison
		everyoneGroup, err := userAdminClient.Group(ctx, user.OrganizationID, nicloudsdk.GroupRequest{})
		require.NoError(t, err)
		normalizeGroupMembers(&everyoneGroup)

		groups, err := userAdminClient.Groups(ctx, nicloudsdk.GroupArguments{
			Organization: user.OrganizationID.String(),
		})
		require.NoError(t, err)
		normalizeAllGroups(groups)

		// 'Everyone' group + 2 custom groups.
		require.ElementsMatch(t, []nicloudsdk.Group{
			everyoneGroup,
			group1,
			group2,
		}, groups)

		// Filter by user
		user5Groups, err := userAdminClient.Groups(ctx, nicloudsdk.GroupArguments{
			HasMember: user5.Username,
		})
		require.NoError(t, err)
		normalizeAllGroups(user5Groups)
		// Everyone group and group 2
		require.ElementsMatch(t, []nicloudsdk.Group{
			everyoneGroup,
			group2,
		}, user5Groups)

		// Query from the user's perspective
		user5View, err := user5Client.Groups(ctx, nicloudsdk.GroupArguments{})
		require.NoError(t, err)
		normalizeAllGroups(user5View)

		// Org members can read all groups when workspace sharing is not
		// disabled, but group membership is limited to the requesting user.
		// TODO(geokat): add another test with workspace sharing disabled.
		require.Len(t, user5View, 3)
		user5ViewIDs := slice.List(user5View, func(g nicloudsdk.Group) uuid.UUID {
			return g.ID
		})

		require.ElementsMatch(t, []uuid.UUID{
			everyoneGroup.ID,
			group1.ID,
			group2.ID,
		}, user5ViewIDs)
		for _, g := range user5View {
			if g.ID == everyoneGroup.ID || g.ID == group2.ID {
				// Only expect the 1 member, themselves.
				require.Len(t, g.Members, 1)
				require.Equal(t, user5.ReducedUser.ID, g.Members[0].MinimalUser.ID)
				continue
			}

			require.Empty(t, g.Members)
		}
	})
}

func TestDeleteGroup(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		group1, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		err = userAdminClient.DeleteGroup(ctx, group1.ID)
		require.NoError(t, err)

		_, err = userAdminClient.Group(ctx, group1.ID, nicloudsdk.GroupRequest{})
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, cerr.StatusCode())
	})

	t.Run("Audit", func(t *testing.T) {
		t.Parallel()

		auditor := audit.NewMock()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			AuditLogging: true,
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				Auditor:                  auditor,
			},
		})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

		_ = nicloudenttest.AddLicense(t, client, nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
				nicloudsdk.FeatureAuditLog:     1,
			},
		})
		ctx := testutil.Context(t, testutil.WaitLong)

		group, err := userAdminClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "hi",
		})
		require.NoError(t, err)

		numLogs := len(auditor.AuditLogs())
		err = userAdminClient.DeleteGroup(ctx, group.ID)
		require.NoError(t, err)
		numLogs++

		require.Len(t, auditor.AuditLogs(), numLogs)
		require.True(t, auditor.Contains(t, database.AuditLog{
			Action:     database.AuditActionDelete,
			ResourceID: group.ID,
		}))
	})

	t.Run("allUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())
		ctx := testutil.Context(t, testutil.WaitLong)
		err := userAdminClient.DeleteGroup(ctx, user.OrganizationID)
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})
}

func TestGetGroupMembersFilter(t *testing.T) {
	t.Parallel()

	client, db, first := nicloudenttest.NewWithDatabase(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
			OIDCConfig: &nicloud.OIDCConfig{
				AllowSignups: true,
			},
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC:    1,
				nicloudsdk.FeatureServiceAccounts: 1,
			},
		},
	})

	userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleUserAdmin())

	setupCtx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	t.Cleanup(cancel)

	group, err := userAdminClient.CreateGroup(setupCtx, first.OrganizationID, nicloudsdk.CreateGroupRequest{
		Name: "filtered",
	})
	require.NoError(t, err)

	setup := func(users []nicloudsdk.User) {
		userIDs := make([]string, len(users))
		for i, user := range users {
			userIDs[i] = user.ID.String()
		}
		group, err = userAdminClient.PatchGroup(setupCtx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: userIDs,
		})
		require.NoError(t, err)
	}
	fetch := func(testCtx context.Context, req nicloudsdk.UsersRequest) []nicloudsdk.ReducedUser {
		res, err := userAdminClient.GroupMembers(testCtx, group.ID, req)
		require.NoError(t, err)
		return res.Users
	}
	options := &nicloudtest.UsersFilterOptions{CreateServiceAccounts: true}
	nicloudtest.UsersFilter(setupCtx, t, client, db, options, setup, fetch)
}

func TestGetGroupMembersPagination(t *testing.T) {
	t.Parallel()

	client, first := nicloudenttest.New(t, &nicloudenttest.Options{
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		},
	})

	userAdminClient, _ := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleUserAdmin())

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	t.Cleanup(cancel)

	group, err := userAdminClient.CreateGroup(ctx, first.OrganizationID, nicloudsdk.CreateGroupRequest{
		Name: "paginated",
	})
	require.NoError(t, err)

	setup := func(users []nicloudsdk.User) {
		userIDs := make([]string, len(users))
		for i, user := range users {
			userIDs[i] = user.ID.String()
		}
		group, err = userAdminClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: userIDs,
		})
		require.NoError(t, err)
	}
	fetch := func(req nicloudsdk.UsersRequest) ([]nicloudsdk.ReducedUser, int) {
		group, err := userAdminClient.GroupMembers(ctx, group.ID, req)
		require.NoError(t, err)
		return group.Users, group.Count
	}
	nicloudtest.UsersPagination(ctx, t, client, setup, fetch)
}
