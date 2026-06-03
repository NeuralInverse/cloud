package nicloud_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestWorkspaceBuild(t *testing.T) {
	t.Parallel()

	// Only use this context for setup. Use a separate context for subtests!
	setupCtx := testutil.Context(t, testutil.WaitMedium)
	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureAccessControl:              1,
				nicloudsdk.FeatureTemplateRBAC:               1,
				nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
			},
		},
	})

	// For this test we create two templates:
	// tplA will be used to test creation of new workspaces.
	// tplB will be used to test builds on existing workspaces.
	// This is done to enable parallelization of the sub-tests without them interfering with each other.
	// Both templates mandate the promoted version.
	// This should be enforced for everyone except template admins.
	tplAv1 := nicloudtest.CreateTemplateVersion(t, ownerClient, owner.OrganizationID, nil)
	tplA := nicloudtest.CreateTemplate(t, ownerClient, owner.OrganizationID, tplAv1.ID)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, ownerClient, tplAv1.ID)
	require.Equal(t, tplAv1.ID, tplA.ActiveVersionID)
	tplA = nicloudtest.UpdateTemplateMeta(t, ownerClient, tplA.ID, nicloudsdk.UpdateTemplateMeta{
		RequireActiveVersion: ptr.Ref(true),
	})
	require.True(t, tplA.RequireActiveVersion)
	tplAv2 := nicloudtest.CreateTemplateVersion(t, ownerClient, owner.OrganizationID, nil, func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
		ctvr.TemplateID = tplA.ID
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, ownerClient, tplAv2.ID)
	nicloudtest.UpdateActiveTemplateVersion(t, ownerClient, tplA.ID, tplAv2.ID)

	tplBv1 := nicloudtest.CreateTemplateVersion(t, ownerClient, owner.OrganizationID, nil)
	tplB := nicloudtest.CreateTemplate(t, ownerClient, owner.OrganizationID, tplBv1.ID)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, ownerClient, tplBv1.ID)
	require.Equal(t, tplBv1.ID, tplB.ActiveVersionID)
	tplB = nicloudtest.UpdateTemplateMeta(t, ownerClient, tplB.ID, nicloudsdk.UpdateTemplateMeta{
		RequireActiveVersion: ptr.Ref(true),
	})
	require.True(t, tplB.RequireActiveVersion)

	templateAdminClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
	templateACLAdminClient, templateACLAdmin := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)
	templateGroupACLAdminClient, templateGroupACLAdmin := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)
	memberClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

	// Create a group so we can also test group template admin ownership.
	// Add the user who gains template admin via group membership.
	group := nicloudtest.CreateGroup(t, ownerClient, owner.OrganizationID, "test", templateGroupACLAdmin)

	// Update the template for both users and groups.
	//nolint:gocritic // test setup
	for _, tpl := range []nicloudsdk.Template{tplA, tplB} {
		err := ownerClient.UpdateTemplateACL(setupCtx, tpl.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				templateACLAdmin.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
			GroupPerms: map[string]nicloudsdk.TemplateRole{
				group.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		})
		require.NoError(t, err, "updating template ACL for template %q", tpl.ID)
	}

	type testcase struct {
		Name               string
		Client             *nicloudsdk.Client
		ExpectedStatusCode int
	}

	cases := []testcase{
		{
			Name:               "OwnerOK",
			Client:             ownerClient,
			ExpectedStatusCode: http.StatusOK,
		},
		{
			Name:               "TemplateAdminOK",
			Client:             templateAdminClient,
			ExpectedStatusCode: http.StatusOK,
		},
		{
			Name:               "TemplateACLAdminOK",
			Client:             templateACLAdminClient,
			ExpectedStatusCode: http.StatusOK,
		},
		{
			Name:               "TemplateGroupACLAdminOK",
			Client:             templateGroupACLAdminClient,
			ExpectedStatusCode: http.StatusOK,
		},
		{
			Name:               "MemberFailsToCreate",
			Client:             memberClient,
			ExpectedStatusCode: http.StatusForbidden,
		},
	}

	// Create pre-existing workspaces for each of the test cases.
	var extantWorkspaces []nicloudsdk.Workspace
	for _, c := range cases {
		extantWs, err := c.Client.CreateUserWorkspace(setupCtx, nicloudsdk.Me, nicloudsdk.CreateWorkspaceRequest{
			TemplateVersionID: tplB.ActiveVersionID,
			Name:              testutil.GetRandomNameHyphenated(t),
			AutomaticUpdates:  nicloudsdk.AutomaticUpdatesNever,
		})
		require.NoError(t, err, "setup workspace for case %q", c.Name)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, c.Client, extantWs.LatestBuild.ID)
		extantWorkspaces = append(extantWorkspaces, extantWs)
	}

	// Create a new version of template B and promote it to be the active version.
	tplBv2 := nicloudtest.CreateTemplateVersion(t, ownerClient, owner.OrganizationID, nil, func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
		ctvr.TemplateID = tplB.ID
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, ownerClient, tplBv2.ID)
	nicloudtest.UpdateActiveTemplateVersion(t, ownerClient, tplB.ID, tplBv2.ID)

	t.Run("NewWorkspace", func(t *testing.T) {
		t.Parallel()

		for _, c := range cases {
			t.Run(c.Name, func(t *testing.T) {
				t.Parallel()
				ctx := testutil.Context(t, testutil.WaitMedium)
				ws, err := c.Client.CreateUserWorkspace(ctx, nicloudsdk.Me, nicloudsdk.CreateWorkspaceRequest{
					TemplateVersionID: tplAv1.ID,
					Name:              testutil.GetRandomNameHyphenated(t),
					AutomaticUpdates:  nicloudsdk.AutomaticUpdatesNever,
				})
				if c.ExpectedStatusCode == http.StatusOK {
					require.NoError(t, err)
					require.Equal(t, tplAv1.ID, ws.LatestBuild.TemplateVersionID, "workspace did not use expected version for case %q", c.Name)
				} else {
					require.Error(t, err)
					cerr, ok := nicloudsdk.AsError(err)
					require.True(t, ok)
					require.Equal(t, c.ExpectedStatusCode, cerr.StatusCode())
				}
			})
		}
	})

	t.Run("ExistingWorkspace", func(t *testing.T) {
		t.Parallel()

		for idx, c := range cases {
			t.Run(c.Name, func(t *testing.T) {
				t.Parallel()
				ctx := testutil.Context(t, testutil.WaitMedium)
				// Stopping the workspace must always succeed.
				wb, err := c.Client.CreateWorkspaceBuild(ctx, extantWorkspaces[idx].ID, nicloudsdk.CreateWorkspaceBuildRequest{
					Transition: nicloudsdk.WorkspaceTransitionStop,
				})
				require.NoError(t, err, "stopping workspace for case %q", c.Name)
				nicloudtest.AwaitWorkspaceBuildJobCompleted(t, c.Client, wb.ID)

				// Attempt to start the workspace with the given version.
				wb, err = c.Client.CreateWorkspaceBuild(ctx, extantWorkspaces[idx].ID, nicloudsdk.CreateWorkspaceBuildRequest{
					Transition:        nicloudsdk.WorkspaceTransitionStart,
					TemplateVersionID: tplBv1.ID,
				})
				if c.ExpectedStatusCode == http.StatusOK {
					require.NoError(t, err, "starting workspace for case %q", c.Name)
					nicloudtest.AwaitWorkspaceBuildJobCompleted(t, c.Client, wb.ID)
					require.Equal(t, tplBv1.ID, wb.TemplateVersionID, "workspace did not use expected version for case %q", c.Name)
				} else {
					require.Error(t, err, "starting workspace for case %q", c.Name)
					cerr, ok := nicloudsdk.AsError(err)
					require.True(t, ok)
					require.Equal(t, c.ExpectedStatusCode, cerr.StatusCode())
				}
			})
		}
	})
}
