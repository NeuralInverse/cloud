package nicloud_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog/v3"
	"cdr.dev/slog/v3/sloggers/slogtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications/notificationstest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/cryptorand"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/schedule"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestTemplates(t *testing.T) {
	t.Parallel()

	logger := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true}).Leveled(slog.LevelDebug)

	t.Run("Deprecated", func(t *testing.T) {
		t.Parallel()

		notifyEnq := &notificationstest.FakeEnqueuer{}
		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				NotificationsEnqueuer:    notifyEnq,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAccessControl: 1,
				},
			},
		})
		client, secondUser := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.RoleTemplateAdmin())
		otherClient, otherUser := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.RoleTemplateAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		_ = nicloudtest.CreateWorkspace(t, owner, template.ID)
		_ = nicloudtest.CreateWorkspace(t, client, template.ID)

		// Create another template for testing that users of another template do not
		// get a notification.
		secondVersion := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		secondTemplate := nicloudtest.CreateTemplate(t, client, user.OrganizationID, secondVersion.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, secondVersion.ID)

		_ = nicloudtest.CreateWorkspace(t, otherClient, secondTemplate.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		updated, err := client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			DeprecationMessage: ptr.Ref("Stop using this template"),
		})
		require.NoError(t, err)
		assert.Greater(t, updated.UpdatedAt, template.UpdatedAt)
		// AGPL cannot deprecate, expect no change
		assert.True(t, updated.Deprecated)
		assert.NotEmpty(t, updated.DeprecationMessage)

		notifs := []*notificationstest.FakeNotification{}
		for _, notif := range notifyEnq.Sent() {
			if notif.TemplateID == notifications.TemplateTemplateDeprecated {
				notifs = append(notifs, notif)
			}
		}
		require.Equal(t, 2, len(notifs))

		expectedSentTo := []string{user.UserID.String(), secondUser.ID.String()}
		slices.Sort(expectedSentTo)

		sentTo := []string{}
		for _, notif := range notifs {
			sentTo = append(sentTo, notif.UserID.String())
		}
		slices.Sort(sentTo)

		// Require the notification to have only been sent to the expected users
		assert.Equal(t, expectedSentTo, sentTo)

		// The previous check should verify this but we're double checking that
		// the notification wasn't sent to users not using the template.
		for _, notif := range notifs {
			assert.NotEqual(t, otherUser.ID, notif.UserID)
		}

		_, err = client.CreateWorkspace(ctx, user.OrganizationID, nicloudsdk.Me, nicloudsdk.CreateWorkspaceRequest{
			TemplateID: template.ID,
			Name:       "foobar",
		})
		require.ErrorContains(t, err, "deprecated")

		// Unset deprecated and try again
		updated, err = client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{DeprecationMessage: ptr.Ref("")})
		require.NoError(t, err)
		assert.False(t, updated.Deprecated)
		assert.Empty(t, updated.DeprecationMessage)

		_, err = client.CreateWorkspace(ctx, user.OrganizationID, nicloudsdk.Me, nicloudsdk.CreateWorkspaceRequest{
			TemplateID: template.ID,
			Name:       "foobar",
		})
		require.NoError(t, err)
	})

	t.Run("MaxPortShareLevel", func(t *testing.T) {
		t.Parallel()

		cfg := nicloudtest.DeploymentValues(t)
		cfg.Experiments = []string{"shared-ports"}
		owner, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				DeploymentValues:         cfg,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureControlSharedPorts: 1,
				},
			},
		})
		client, _ := nicloudtest.CreateAnotherUser(t, owner, user.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:         echo.ParseComplete,
			ProvisionPlan: echo.PlanComplete,
			ProvisionGraph: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_INFO,
						Output: "example",
					},
				},
			}, {
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Resources: []*proto.Resource{{
							Name: "some",
							Type: "example",
							Agents: []*proto.Agent{{
								Id:   "something",
								Name: "test",
								Auth: &proto.Agent_Token{
									Token: uuid.NewString(),
								},
							}},
						}, {
							Name: "another",
							Type: "example",
						}},
					},
				},
			}},
		})
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID, func(ctr *nicloudsdk.CreateTemplateRequest) {
			ctr.MaxPortShareLevel = ptr.Ref(nicloudsdk.WorkspaceAgentPortShareLevelPublic)
		})
		require.Equal(t, template.MaxPortShareLevel, nicloudsdk.WorkspaceAgentPortShareLevelPublic)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		ws := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, ws.LatestBuild.ID)
		ws, err := client.Workspace(context.Background(), ws.ID)
		require.NoError(t, err)

		ctx := testutil.Context(t, testutil.WaitLong)

		// OK: setting the same level is a no-op under the new PATCH semantics
		// (304 Not Modified) but must not be a server error.
		var level nicloudsdk.WorkspaceAgentPortShareLevel = nicloudsdk.WorkspaceAgentPortShareLevelPublic
		_, err = client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			MaxPortShareLevel: &level,
		})
		require.NoError(t, err)
		template, err = client.Template(ctx, template.ID)
		require.NoError(t, err)
		assert.Equal(t, level, template.MaxPortShareLevel)
		// Invalid level
		level = "invalid"
		_, err = client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			MaxPortShareLevel: &level,
		})
		require.ErrorContains(t, err, "invalid max port sharing level")

		// Create public port share
		_, err = client.UpsertWorkspaceAgentPortShare(ctx, ws.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
			AgentName:  ws.LatestBuild.Resources[0].Agents[0].Name,
			Port:       8080,
			ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
			Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
		})
		require.NoError(t, err)

		// Reduce max level to authenticated
		level = nicloudsdk.WorkspaceAgentPortShareLevelAuthenticated
		_, err = client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			MaxPortShareLevel: &level,
		})
		require.NoError(t, err)

		// Ensure previously public port is now authenticated
		wpsr, err := client.GetWorkspaceAgentPortShares(ctx, ws.ID)
		require.NoError(t, err)
		require.Len(t, wpsr.Shares, 1)
		assert.Equal(t, nicloudsdk.WorkspaceAgentPortShareLevelAuthenticated, wpsr.Shares[0].ShareLevel)

		// reduce max level to owner
		level = nicloudsdk.WorkspaceAgentPortShareLevelOwner
		_, err = client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			MaxPortShareLevel: &level,
		})
		require.NoError(t, err)

		// Ensure previously authenticated port is removed
		wpsr, err = client.GetWorkspaceAgentPortShares(ctx, ws.ID)
		require.NoError(t, err)
		require.Empty(t, wpsr.Shares)
	})

	t.Run("SetAutostartRequirement", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		require.Equal(t, []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}, template.AutostartRequirement.DaysOfWeek)

		ctx := testutil.Context(t, testutil.WaitLong)
		updated, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			Name:        ptr.Ref(template.Name),
			DisplayName: &template.DisplayName,
			Description: &template.Description,
			Icon:        &template.Icon,
			AutostartRequirement: &nicloudsdk.TemplateAutostartRequirement{
				DaysOfWeek: []string{"monday", "saturday"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"monday", "saturday"}, updated.AutostartRequirement.DaysOfWeek)

		template, err = anotherClient.Template(ctx, template.ID)
		require.NoError(t, err)
		require.Equal(t, []string{"monday", "saturday"}, template.AutostartRequirement.DaysOfWeek)

		// Ensure a missing field is a noop
		updated, err = anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			Name:        ptr.Ref(template.Name),
			DisplayName: &template.DisplayName,
			Description: &template.Description,
			Icon:        ptr.Ref(template.Icon + "something"),
		})
		require.NoError(t, err)
		require.Equal(t, []string{"monday", "saturday"}, updated.AutostartRequirement.DaysOfWeek)

		template, err = anotherClient.Template(ctx, template.ID)
		require.NoError(t, err)
		require.Equal(t, []string{"monday", "saturday"}, template.AutostartRequirement.DaysOfWeek)
		require.Empty(t, template.DeprecationMessage)
		require.False(t, template.Deprecated)
	})

	t.Run("SetInvalidAutostartRequirement", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		require.Equal(t, []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}, template.AutostartRequirement.DaysOfWeek)

		ctx := testutil.Context(t, testutil.WaitLong)
		_, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			Name:        ptr.Ref(template.Name),
			DisplayName: &template.DisplayName,
			Description: &template.Description,
			Icon:        &template.Icon,
			AutostartRequirement: &nicloudsdk.TemplateAutostartRequirement{
				DaysOfWeek: []string{"foobar", "saturday"},
			},
		})
		require.Error(t, err)
		require.Empty(t, template.DeprecationMessage)
		require.False(t, template.Deprecated)
	})

	t.Run("SetAutostopRequirement", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		require.Empty(t, 0, template.AutostopRequirement.DaysOfWeek)
		require.EqualValues(t, 1, template.AutostopRequirement.Weeks)

		ctx := context.Background()
		updated, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			Name:                         ptr.Ref(template.Name),
			DisplayName:                  &template.DisplayName,
			Description:                  &template.Description,
			Icon:                         &template.Icon,
			AllowUserCancelWorkspaceJobs: ptr.Ref(template.AllowUserCancelWorkspaceJobs),
			DefaultTTLMillis:             ptr.Ref(time.Hour.Milliseconds()),
			AutostopRequirement: &nicloudsdk.TemplateAutostopRequirement{
				DaysOfWeek: []string{"monday", "saturday"},
				Weeks:      3,
			},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"monday", "saturday"}, updated.AutostopRequirement.DaysOfWeek)
		require.EqualValues(t, 3, updated.AutostopRequirement.Weeks)

		template, err = anotherClient.Template(ctx, template.ID)
		require.NoError(t, err)
		require.Equal(t, []string{"monday", "saturday"}, template.AutostopRequirement.DaysOfWeek)
		require.EqualValues(t, 3, template.AutostopRequirement.Weeks)
		require.Empty(t, template.DeprecationMessage)
		require.False(t, template.Deprecated)
	})

	t.Run("CleanupTTLs", func(t *testing.T) {
		t.Run("OK", func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t, testutil.WaitMedium)
			client, user := nicloudenttest.New(t, &nicloudenttest.Options{
				Options: &nicloudtest.Options{
					IncludeProvisionerDaemon: true,
				},
				LicenseOptions: &nicloudenttest.LicenseOptions{
					Features: license.Features{
						nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
					},
				},
			})
			anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

			version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
			nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
			template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
			require.EqualValues(t, 0, template.TimeTilDormantMillis)
			require.EqualValues(t, 0, template.FailureTTLMillis)
			require.EqualValues(t, 0, template.TimeTilDormantAutoDeleteMillis)

			var (
				failureTTL    = 1 * time.Minute
				inactivityTTL = 2 * time.Minute
				dormantTTL    = 3 * time.Minute
			)

			updated, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
				Name:                           ptr.Ref(template.Name),
				DisplayName:                    &template.DisplayName,
				Description:                    &template.Description,
				Icon:                           &template.Icon,
				AllowUserCancelWorkspaceJobs:   ptr.Ref(template.AllowUserCancelWorkspaceJobs),
				TimeTilDormantMillis:           ptr.Ref(inactivityTTL.Milliseconds()),
				FailureTTLMillis:               ptr.Ref(failureTTL.Milliseconds()),
				TimeTilDormantAutoDeleteMillis: ptr.Ref(dormantTTL.Milliseconds()),
			})
			require.NoError(t, err)
			require.Equal(t, failureTTL.Milliseconds(), updated.FailureTTLMillis)
			require.Equal(t, inactivityTTL.Milliseconds(), updated.TimeTilDormantMillis)
			require.Equal(t, dormantTTL.Milliseconds(), updated.TimeTilDormantAutoDeleteMillis)

			// Validate fetching the template returns the same values as updating
			// the template.
			template, err = anotherClient.Template(ctx, template.ID)
			require.NoError(t, err)
			require.Equal(t, failureTTL.Milliseconds(), updated.FailureTTLMillis)
			require.Equal(t, inactivityTTL.Milliseconds(), updated.TimeTilDormantMillis)
			require.Equal(t, dormantTTL.Milliseconds(), updated.TimeTilDormantAutoDeleteMillis)
		})

		t.Run("BadRequest", func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t, testutil.WaitMedium)
			client, user := nicloudenttest.New(t, &nicloudenttest.Options{
				Options: &nicloudtest.Options{
					IncludeProvisionerDaemon: true,
				},
				LicenseOptions: &nicloudenttest.LicenseOptions{
					Features: license.Features{
						nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
					},
				},
			})
			anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

			version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
			nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
			template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

			type testcase struct {
				Name                string
				TimeTilDormantMS    int64
				FailureTTLMS        int64
				DormantAutoDeleteMS int64
			}

			cases := []testcase{
				{
					Name:                "NegativeValue",
					TimeTilDormantMS:    -1,
					FailureTTLMS:        -2,
					DormantAutoDeleteMS: -3,
				},
				{
					Name:                "ValueTooSmall",
					TimeTilDormantMS:    1,
					FailureTTLMS:        999,
					DormantAutoDeleteMS: 500,
				},
			}

			for _, c := range cases {
				// nolint: paralleltest // context is from parent t.Run
				t.Run(c.Name, func(t *testing.T) {
					_, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
						Name:                           ptr.Ref(template.Name),
						DisplayName:                    &template.DisplayName,
						Description:                    &template.Description,
						Icon:                           &template.Icon,
						AllowUserCancelWorkspaceJobs:   ptr.Ref(template.AllowUserCancelWorkspaceJobs),
						TimeTilDormantMillis:           ptr.Ref(c.TimeTilDormantMS),
						FailureTTLMillis:               ptr.Ref(c.FailureTTLMS),
						TimeTilDormantAutoDeleteMillis: ptr.Ref(c.DormantAutoDeleteMS),
					})
					require.Error(t, err)
					cerr, ok := nicloudsdk.AsError(err)
					require.True(t, ok)
					require.Len(t, cerr.Validations, 3)
					require.Equal(t, "Value must be at least one minute.", cerr.Validations[0].Detail)
				})
			}
		})
	})

	t.Run("UpdateTimeTilDormantAutoDelete", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t, testutil.WaitMedium)
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		activeWS := nicloudtest.CreateWorkspace(t, anotherClient, template.ID)
		dormantWS := nicloudtest.CreateWorkspace(t, anotherClient, template.ID)
		require.Nil(t, activeWS.DeletingAt)
		require.Nil(t, dormantWS.DeletingAt)

		_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, activeWS.LatestBuild.ID)
		_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, dormantWS.LatestBuild.ID)

		err := anotherClient.UpdateWorkspaceDormancy(ctx, dormantWS.ID, nicloudsdk.UpdateWorkspaceDormancy{
			Dormant: true,
		})
		require.NoError(t, err)

		dormantWS = nicloudtest.MustWorkspace(t, client, dormantWS.ID)
		require.NotNil(t, dormantWS.DormantAt)
		// The deleting_at field should be nil since there is no template time_til_dormant_autodelete set.
		require.Nil(t, dormantWS.DeletingAt)

		dormantTTL := time.Minute
		updated, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			TimeTilDormantAutoDeleteMillis: ptr.Ref(dormantTTL.Milliseconds()),
		})
		require.NoError(t, err)
		require.Equal(t, dormantTTL.Milliseconds(), updated.TimeTilDormantAutoDeleteMillis)

		activeWS = nicloudtest.MustWorkspace(t, client, activeWS.ID)
		require.Nil(t, activeWS.DormantAt)
		require.Nil(t, activeWS.DeletingAt)

		updatedDormantWorkspace := nicloudtest.MustWorkspace(t, client, dormantWS.ID)
		require.NotNil(t, updatedDormantWorkspace.DormantAt)
		require.NotNil(t, updatedDormantWorkspace.DeletingAt)
		require.Equal(t, updatedDormantWorkspace.DormantAt.Add(dormantTTL), *updatedDormantWorkspace.DeletingAt)
		require.Equal(t, updatedDormantWorkspace.DormantAt, dormantWS.DormantAt)

		// Disable the time_til_dormant_auto_delete on the template, then we can assert that the workspaces
		// no longer have a deleting_at field.
		updated, err = anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			TimeTilDormantAutoDeleteMillis: ptr.Ref[int64](0),
		})
		require.NoError(t, err)
		require.EqualValues(t, 0, updated.TimeTilDormantAutoDeleteMillis)

		// The active workspace should remain unchanged.
		activeWS = nicloudtest.MustWorkspace(t, client, activeWS.ID)
		require.Nil(t, activeWS.DormantAt)
		require.Nil(t, activeWS.DeletingAt)

		// Fetch the dormant workspace. It should still be dormant, but it should no
		// longer be scheduled for deletion.
		dormantWS = nicloudtest.MustWorkspace(t, client, dormantWS.ID)
		require.NotNil(t, dormantWS.DormantAt)
		require.Nil(t, dormantWS.DeletingAt)
	})

	t.Run("UpdateDormantAt", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t, testutil.WaitMedium)
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		activeWS := nicloudtest.CreateWorkspace(t, anotherClient, template.ID)
		dormantWS := nicloudtest.CreateWorkspace(t, anotherClient, template.ID)
		require.Nil(t, activeWS.DeletingAt)
		require.Nil(t, dormantWS.DeletingAt)

		_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, activeWS.LatestBuild.ID)
		_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, dormantWS.LatestBuild.ID)

		err := anotherClient.UpdateWorkspaceDormancy(ctx, dormantWS.ID, nicloudsdk.UpdateWorkspaceDormancy{
			Dormant: true,
		})
		require.NoError(t, err)

		dormantWS = nicloudtest.MustWorkspace(t, client, dormantWS.ID)
		require.NotNil(t, dormantWS.DormantAt)
		// The deleting_at field should be nil since there is no template time_til_dormant_autodelete set.
		require.Nil(t, dormantWS.DeletingAt)

		dormantTTL := time.Minute
		//nolint:gocritic // non-template-admin cannot update template meta
		updated, err := client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			TimeTilDormantAutoDeleteMillis: ptr.Ref(dormantTTL.Milliseconds()),
			UpdateWorkspaceDormantAt:       ptr.Ref(true),
		})
		require.NoError(t, err)
		require.Equal(t, dormantTTL.Milliseconds(), updated.TimeTilDormantAutoDeleteMillis)

		activeWS = nicloudtest.MustWorkspace(t, client, activeWS.ID)
		require.Nil(t, activeWS.DormantAt)
		require.Nil(t, activeWS.DeletingAt)

		updatedDormantWorkspace := nicloudtest.MustWorkspace(t, client, dormantWS.ID)
		require.NotNil(t, updatedDormantWorkspace.DormantAt)
		require.NotNil(t, updatedDormantWorkspace.DeletingAt)
		// Validate that the workspace dormant_at value is updated.
		require.True(t, updatedDormantWorkspace.DormantAt.After(*dormantWS.DormantAt))
		require.Equal(t, updatedDormantWorkspace.DormantAt.Add(dormantTTL), *updatedDormantWorkspace.DeletingAt)
	})

	t.Run("UpdateLastUsedAt", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t, testutil.WaitMedium)
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAdvancedTemplateScheduling: 1,
				},
			},
		})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		activeWorkspace := nicloudtest.CreateWorkspace(t, anotherClient, template.ID)
		dormantWorkspace := nicloudtest.CreateWorkspace(t, anotherClient, template.ID)
		require.Nil(t, activeWorkspace.DeletingAt)
		require.Nil(t, dormantWorkspace.DeletingAt)

		_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, activeWorkspace.LatestBuild.ID)
		_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, dormantWorkspace.LatestBuild.ID)

		err := anotherClient.UpdateWorkspaceDormancy(ctx, dormantWorkspace.ID, nicloudsdk.UpdateWorkspaceDormancy{
			Dormant: true,
		})
		require.NoError(t, err)

		dormantWorkspace = nicloudtest.MustWorkspace(t, client, dormantWorkspace.ID)
		require.NotNil(t, dormantWorkspace.DormantAt)
		// The deleting_at field should be nil since there is no template time_til_dormant_autodelete set.
		require.Nil(t, dormantWorkspace.DeletingAt)

		inactivityTTL := time.Minute
		updated, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			TimeTilDormantMillis:      ptr.Ref(inactivityTTL.Milliseconds()),
			UpdateWorkspaceLastUsedAt: ptr.Ref(true),
		})
		require.NoError(t, err)
		require.Equal(t, inactivityTTL.Milliseconds(), updated.TimeTilDormantMillis)

		updatedActiveWS := nicloudtest.MustWorkspace(t, client, activeWorkspace.ID)
		require.Nil(t, updatedActiveWS.DormantAt)
		require.Nil(t, updatedActiveWS.DeletingAt)
		require.True(t, updatedActiveWS.LastUsedAt.After(activeWorkspace.LastUsedAt))

		updatedDormantWS := nicloudtest.MustWorkspace(t, client, dormantWorkspace.ID)
		require.NotNil(t, updatedDormantWS.DormantAt)
		require.Nil(t, updatedDormantWS.DeletingAt)
		// Validate that the workspace dormant_at value is updated.
		require.Equal(t, updatedDormantWS.DormantAt, dormantWorkspace.DormantAt)
		require.True(t, updatedDormantWS.LastUsedAt.After(dormantWorkspace.LastUsedAt))
	})

	t.Run("RequireActiveVersion", func(t *testing.T) {
		t.Parallel()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				TemplateScheduleStore:    schedule.NewEnterpriseTemplateScheduleStore(agplUserQuietHoursScheduleStore(), notifications.NewNoopEnqueuer(), logger, nil),
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAccessControl: 1,
				},
			},
		})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID, func(ctr *nicloudsdk.CreateTemplateRequest) {
			ctr.RequireActiveVersion = true
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		require.True(t, template.RequireActiveVersion)

		ctx := testutil.Context(t, testutil.WaitLong)

		// Update the field and assert it persists.
		updatedTemplate, err := anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			RequireActiveVersion: ptr.Ref(false),
		})
		require.NoError(t, err)
		require.False(t, updatedTemplate.RequireActiveVersion)

		// Flip it back to ensure we aren't hardcoding to a default value.
		updatedTemplate, err = anotherClient.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			RequireActiveVersion: ptr.Ref(true),
		})
		require.NoError(t, err)
		require.True(t, updatedTemplate.RequireActiveVersion)

		// Assert that fetching a template is no different from the response
		// when updating.
		template, err = anotherClient.Template(ctx, template.ID)
		require.NoError(t, err)
		require.Equal(t, updatedTemplate, template)
		require.Empty(t, template.DeprecationMessage)
		require.False(t, template.Deprecated)
	})

	// Create a template, remove the group, see if an owner can
	// still fetch the template.
	t.Run("GetOnEveryoneRemove", func(t *testing.T) {
		t.Parallel()
		owner, first := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
				TemplateScheduleStore:    schedule.NewEnterpriseTemplateScheduleStore(agplUserQuietHoursScheduleStore(), notifications.NewNoopEnqueuer(), logger, nil),
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAccessControl: 1,
					nicloudsdk.FeatureTemplateRBAC:  1,
				},
			},
		})

		client, _ := nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitMedium)
		err := client.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: nil,
			GroupPerms: map[string]nicloudsdk.TemplateRole{
				// OrgID is the everyone ID
				first.OrganizationID.String(): nicloudsdk.TemplateRoleDeleted,
			},
		})
		require.NoError(t, err)

		_, err = owner.Template(ctx, template.ID)
		require.NoError(t, err)
	})

	// Create a template in a second organization via custom role
	t.Run("SecondOrganization", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		ownerClient, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues:         dv,
				IncludeProvisionerDaemon: false,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAccessControl:              1,
					nicloudsdk.FeatureCustomRoles:                1,
					nicloudsdk.FeatureExternalProvisionerDaemons: 1,
					nicloudsdk.FeatureMultipleOrganizations:      1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitMedium)
		secondOrg := nicloudenttest.CreateOrganization(t, ownerClient, nicloudenttest.CreateOrganizationOptions{
			IncludeProvisionerDaemon: true,
		})

		//nolint:gocritic // owner required to make custom roles
		orgTemplateAdminRole, err := ownerClient.CreateOrganizationRole(ctx, nicloudsdk.Role{
			Name:           "org-template-admin",
			OrganizationID: secondOrg.ID.String(),
			OrganizationPermissions: nicloudsdk.CreatePermissions(map[nicloudsdk.RBACResource][]nicloudsdk.RBACAction{
				nicloudsdk.ResourceTemplate: nicloudsdk.RBACResourceActions[nicloudsdk.ResourceTemplate],
			}),
		})
		require.NoError(t, err, "create admin role")

		orgTemplateAdmin, _ := nicloudtest.CreateAnotherUser(t, ownerClient, secondOrg.ID, rbac.RoleIdentifier{
			Name:           orgTemplateAdminRole.Name,
			OrganizationID: secondOrg.ID,
		})

		version := nicloudtest.CreateTemplateVersion(t, orgTemplateAdmin, secondOrg.ID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
			ProvisionPlan:  echo.PlanComplete,
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, orgTemplateAdmin, version.ID)

		template := nicloudtest.CreateTemplate(t, orgTemplateAdmin, secondOrg.ID, version.ID)
		require.Equal(t, template.OrganizationID, secondOrg.ID)
	})

	t.Run("MultipleOrganizations", func(t *testing.T) {
		t.Parallel()
		dv := nicloudtest.DeploymentValues(t)
		ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureMultipleOrganizations: 1,
				},
			},
		})

		client, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
		org2 := nicloudenttest.CreateOrganization(t, ownerClient, nicloudenttest.CreateOrganizationOptions{})
		user, _ := nicloudtest.CreateAnotherUser(t, ownerClient, org2.ID)

		// 2 templates in first organization
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		version2 := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version2.ID)

		// 2 in the second organization
		version3 := nicloudtest.CreateTemplateVersion(t, client, org2.ID, nil)
		version4 := nicloudtest.CreateTemplateVersion(t, client, org2.ID, nil)
		nicloudtest.CreateTemplate(t, client, org2.ID, version3.ID)
		nicloudtest.CreateTemplate(t, client, org2.ID, version4.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		// All 4 are viewable by the owner
		templates, err := client.Templates(ctx, nicloudsdk.TemplateFilter{})
		require.NoError(t, err)
		require.Len(t, templates, 4)

		// View a single organization from the owner
		templates, err = client.Templates(ctx, nicloudsdk.TemplateFilter{
			OrganizationID: owner.OrganizationID,
		})
		require.NoError(t, err)
		require.Len(t, templates, 2)

		// Only 2 are viewable by the org user
		templates, err = user.Templates(ctx, nicloudsdk.TemplateFilter{})
		require.NoError(t, err)
		require.Len(t, templates, 2)
		for _, tmpl := range templates {
			require.Equal(t, tmpl.OrganizationName, org2.Name, "organization name on template")
		}
	})
}

func TestTemplateACL(t *testing.T) {
	t.Parallel()

	t.Run("UserRoles", func(t *testing.T) {
		t.Parallel()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		err := anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): nicloudsdk.TemplateRoleUse,
				user3.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		})
		require.NoError(t, err)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		templateUser2 := nicloudsdk.TemplateUser{
			User: user2,
			Role: nicloudsdk.TemplateRoleUse,
		}

		templateUser3 := nicloudsdk.TemplateUser{
			User: user3,
			Role: nicloudsdk.TemplateRoleAdmin,
		}

		require.Len(t, acl.Users, 2)
		require.Contains(t, acl.Users, templateUser2)
		require.Contains(t, acl.Users, templateUser3)
	})

	t.Run("everyoneGroup", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		// Create a user to assert they aren't returned in the response.
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Len(t, acl.Groups, 1)
		require.Len(t, acl.Groups[0].Members, 2) // orgAdmin + TemplateAdmin
		require.Len(t, acl.Users, 0)
	})

	t.Run("NoGroups", func(t *testing.T) {
		t.Parallel()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		client1, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // non-template-admin cannot update template acl
		acl, err := client.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Len(t, acl.Groups, 1)
		require.Len(t, acl.Users, 0)

		// User should be able to read template due to allUsers group.
		_, err = client1.Template(ctx, template.ID)
		require.NoError(t, err)

		allUsers := acl.Groups[0]

		err = client.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			GroupPerms: map[string]nicloudsdk.TemplateRole{
				allUsers.ID.String(): nicloudsdk.TemplateRoleDeleted,
			},
		})
		require.NoError(t, err)

		//nolint:gocritic // non-template-admin cannot update template acl
		acl, err = client.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Len(t, acl.Groups, 0)
		require.Len(t, acl.Users, 0)

		// User should not be able to read template due to allUsers group being deleted.
		_, err = client1.Template(ctx, template.ID)
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, cerr.StatusCode())
	})

	t.Run("DisableEveryoneGroupAccess", func(t *testing.T) {
		t.Parallel()

		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		version := nicloudtest.CreateTemplateVersion(t, client, admin.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, admin.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // non-template-admin cannot get template acl
		acl, err := client.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Equal(t, 1, len(acl.Groups))
		_, err = client.UpdateTemplateMeta(ctx, template.ID, nicloudsdk.UpdateTemplateMeta{
			Name:                         ptr.Ref(template.Name),
			DisplayName:                  &template.DisplayName,
			Description:                  &template.Description,
			Icon:                         &template.Icon,
			AllowUserCancelWorkspaceJobs: ptr.Ref(template.AllowUserCancelWorkspaceJobs),
			DisableEveryoneGroupAccess:   ptr.Ref(true),
		})
		require.NoError(t, err)

		acl, err = client.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Equal(t, 0, len(acl.Groups), acl.Groups)
	})

	// Test that we do not return deleted users.
	t.Run("FilterDeletedUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin(), rbac.RoleUserAdmin())

		_, user1 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		err := anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user1.ID.String(): nicloudsdk.TemplateRoleUse,
			},
		})
		require.NoError(t, err)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Contains(t, acl.Users, nicloudsdk.TemplateUser{
			User: user1,
			Role: nicloudsdk.TemplateRoleUse,
		})

		err = anotherClient.DeleteUser(ctx, user1.ID)
		require.NoError(t, err)

		acl, err = anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Len(t, acl.Users, 0, "deleted users should be filtered")
	})

	// Test that we do not filter dormant users.
	t.Run("IncludeDormantUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin(), rbac.RoleUserAdmin())

		ctx := testutil.Context(t, testutil.WaitLong)

		// nolint:gocritic // Must use owner to create user.
		user1, err := client.CreateUserWithOrgs(ctx, nicloudsdk.CreateUserRequestWithOrgs{
			Email:           "coder@cloud.neuralinverse.com",
			Username:        "coder",
			Password:        "SomeStrongPassword!",
			OrganizationIDs: []uuid.UUID{user.OrganizationID},
		})
		require.NoError(t, err)
		require.Equal(t, nicloudsdk.UserStatusDormant, user1.Status)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		err = anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user1.ID.String(): nicloudsdk.TemplateRoleUse,
			},
		})
		require.NoError(t, err)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Contains(t, acl.Users, nicloudsdk.TemplateUser{
			User: user1,
			Role: nicloudsdk.TemplateRoleUse,
		})
	})

	// Test that we do not return suspended users.
	t.Run("FilterSuspendedUsers", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin(), rbac.RoleUserAdmin())

		_, user1 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		err := anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user1.ID.String(): nicloudsdk.TemplateRoleUse,
			},
		})
		require.NoError(t, err)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Contains(t, acl.Users, nicloudsdk.TemplateUser{
			User: user1,
			Role: nicloudsdk.TemplateRoleUse,
		})

		_, err = anotherClient.UpdateUserStatus(ctx, user1.ID.String(), nicloudsdk.UserStatusSuspended)
		require.NoError(t, err)

		acl, err = anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Len(t, acl.Users, 0, "suspended users should be filtered")
	})

	// Test that we do not return deleted groups.
	t.Run("FilterDeletedGroups", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin(), rbac.RoleUserAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		group, err := anotherClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "test",
		})
		require.NoError(t, err)

		err = anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			GroupPerms: map[string]nicloudsdk.TemplateRole{
				group.ID.String(): nicloudsdk.TemplateRoleUse,
			},
		})
		require.NoError(t, err)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		// Length should be 2 for test group and the implicit allUsers group.
		require.Len(t, acl.Groups, 2)

		require.Contains(t, acl.Groups, nicloudsdk.TemplateGroup{
			Group: group,
			Role:  nicloudsdk.TemplateRoleUse,
		})

		err = anotherClient.DeleteGroup(ctx, group.ID)
		require.NoError(t, err)

		acl, err = anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		// Length should be 1 for the allUsers group.
		require.Len(t, acl.Groups, 1)
		require.NotContains(t, acl.Groups, nicloudsdk.TemplateGroup{
			Group: group,
			Role:  nicloudsdk.TemplateRoleUse,
		})
	})

	t.Run("AdminCanPushVersions", func(t *testing.T) {
		t.Parallel()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		client1, user1 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // test setup
		err := client.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user1.ID.String(): nicloudsdk.TemplateRoleUse,
			},
		})
		require.NoError(t, err)

		data, err := echo.Tar(nil)
		require.NoError(t, err)
		file, err := client1.Upload(context.Background(), nicloudsdk.ContentTypeTar, bytes.NewReader(data))
		require.NoError(t, err)

		_, err = client1.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "testme",
			TemplateID:    template.ID,
			FileID:        file.ID,
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.Error(t, err)

		//nolint:gocritic // test setup
		err = client.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user1.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		})
		require.NoError(t, err)

		_, err = client1.CreateTemplateVersion(ctx, user.OrganizationID, nicloudsdk.CreateTemplateVersionRequest{
			Name:          "testme",
			TemplateID:    template.ID,
			FileID:        file.ID,
			StorageMethod: nicloudsdk.ProvisionerStorageMethodFile,
			Provisioner:   nicloudsdk.ProvisionerTypeEcho,
		})
		require.NoError(t, err)
	})

	// Regression test for PLAT-149. Previously this endpoint did an N+1
	// fetch of every group's members and member count. Verify that the
	// member count is returned correctly for many groups, and that the
	// per-group members list is no longer populated (callers should rely
	// on TotalMemberCount).
	t.Run("AvailableReturnsGroupMemberCounts", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		admin, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin(), rbac.RoleUserAdmin())

		// Create a couple of users we can stuff into groups.
		_, alice := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, bob := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, carol := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)

		// emptyGroup: zero non-system members.
		// singleGroup: alice only.
		// fullGroup: alice + bob + carol.
		emptyGroup := nicloudtest.CreateGroup(t, admin, user.OrganizationID, "empty-group")
		singleGroup := nicloudtest.CreateGroup(t, admin, user.OrganizationID, "single-group", alice)
		fullGroup := nicloudtest.CreateGroup(t, admin, user.OrganizationID, "full-group", alice, bob, carol)

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		available, err := admin.TemplateACLAvailable(ctx, template.ID, nicloudsdk.UsersRequest{})
		require.NoError(t, err)

		wantCounts := map[uuid.UUID]int{
			emptyGroup.ID:  0,
			singleGroup.ID: 1,
			fullGroup.ID:   3,
		}

		found := map[uuid.UUID]bool{}
		for _, group := range available.Groups {
			if want, ok := wantCounts[group.ID]; ok {
				found[group.ID] = true
				require.Equal(t, want, group.TotalMemberCount,
					"unexpected total_member_count for group %q", group.Name)
				require.Empty(t, group.Members,
					"members must not be populated by the available endpoint for group %q", group.Name)
			}
		}
		for id := range wantCounts {
			require.True(t, found[id], "group %s missing from available response", id)
		}
	})

	// Companion to the AvailableReturnsGroupMemberCounts test above. Verifies
	// that the q query parameter applies a server-side substring filter on
	// group name / display_name, and that limit caps the number of groups
	// returned. The autocomplete sends both on each keystroke; before
	// PLAT-149 both were ignored for groups.
	t.Run("AvailableHonorsGroupSearchAndLimit", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		admin, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin(), rbac.RoleUserAdmin())

		// Create a handful of groups with predictable names so we can
		// pin assertions to specific substrings.
		engAlpha := nicloudtest.CreateGroup(t, admin, user.OrganizationID, "engineering-alpha")
		engBeta := nicloudtest.CreateGroup(t, admin, user.OrganizationID, "engineering-beta")
		design := nicloudtest.CreateGroup(t, admin, user.OrganizationID, "design")
		sales := nicloudtest.CreateGroup(t, admin, user.OrganizationID, "sales")

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		groupIDs := func(available nicloudsdk.ACLAvailable) []uuid.UUID {
			ids := make([]uuid.UUID, 0, len(available.Groups))
			for _, g := range available.Groups {
				ids = append(ids, g.ID)
			}
			return ids
		}

		// q filters by group name / display_name substring.
		filtered, err := admin.TemplateACLAvailable(ctx, template.ID, nicloudsdk.UsersRequest{
			SearchQuery: "engineering",
		})
		require.NoError(t, err)
		got := groupIDs(filtered)
		require.ElementsMatch(t, []uuid.UUID{engAlpha.ID, engBeta.ID}, got,
			"q=engineering should return only engineering-* groups, got %v", got)
		require.NotContains(t, got, design.ID)
		require.NotContains(t, got, sales.ID)

		// limit caps the number of groups returned. With 4 user-created
		// groups plus the implicit Everyone group, asking for 2 must
		// return at most 2 groups.
		limited, err := admin.TemplateACLAvailable(ctx, template.ID, nicloudsdk.UsersRequest{
			Pagination: nicloudsdk.Pagination{Limit: 2},
		})
		require.NoError(t, err)
		require.Len(t, limited.Groups, 2,
			"limit=2 should cap groups to 2, got %d", len(limited.Groups))
	})
}

func TestUpdateTemplateACL(t *testing.T) {
	t.Parallel()

	t.Run("UserPerms", func(t *testing.T) {
		t.Parallel()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		err := anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): nicloudsdk.TemplateRoleUse,
				user3.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		})
		require.NoError(t, err)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		templateUser2 := nicloudsdk.TemplateUser{
			User: user2,
			Role: nicloudsdk.TemplateRoleUse,
		}

		templateUser3 := nicloudsdk.TemplateUser{
			User: user3,
			Role: nicloudsdk.TemplateRoleAdmin,
		}

		require.Len(t, acl.Users, 2)
		require.Contains(t, acl.Users, templateUser2)
		require.Contains(t, acl.Users, templateUser3)
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
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		numLogs := len(auditor.AuditLogs())

		req := nicloudsdk.UpdateTemplateACL{
			GroupPerms: map[string]nicloudsdk.TemplateRole{
				user.OrganizationID.String(): nicloudsdk.TemplateRoleDeleted,
			},
		}
		err := anotherClient.UpdateTemplateACL(ctx, template.ID, req)
		require.NoError(t, err)
		numLogs++

		require.Len(t, auditor.AuditLogs(), numLogs)
		require.True(t, auditor.Contains(t, database.AuditLog{
			Action:     database.AuditActionWrite,
			ResourceID: template.ID,
		}))
	})

	t.Run("DeleteUser", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): nicloudsdk.TemplateRoleUse,
				user3.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		}

		ctx := testutil.Context(t, testutil.WaitLong)

		err := anotherClient.UpdateTemplateACL(ctx, template.ID, req)
		require.NoError(t, err)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Contains(t, acl.Users, nicloudsdk.TemplateUser{
			User: user2,
			Role: nicloudsdk.TemplateRoleUse,
		})
		require.Contains(t, acl.Users, nicloudsdk.TemplateUser{
			User: user3,
			Role: nicloudsdk.TemplateRoleAdmin,
		})

		req = nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): nicloudsdk.TemplateRoleAdmin,
				user3.ID.String(): nicloudsdk.TemplateRoleDeleted,
			},
		}

		err = anotherClient.UpdateTemplateACL(ctx, template.ID, req)
		require.NoError(t, err)

		acl, err = anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Contains(t, acl.Users, nicloudsdk.TemplateUser{
			User: user2,
			Role: nicloudsdk.TemplateRoleAdmin,
		})

		require.NotContains(t, acl.Users, nicloudsdk.TemplateUser{
			User: user3,
			Role: nicloudsdk.TemplateRoleAdmin,
		})
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				"hi": nicloudsdk.TemplateRoleAdmin,
			},
		}

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // Testing ACL validation
		err := client.UpdateTemplateACL(ctx, template.ID, req)
		require.Error(t, err)
		cerr, _ := nicloudsdk.AsError(err)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	// We should report invalid UUIDs as errors
	t.Run("DeleteRoleForInvalidUUID", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				"hi": nicloudsdk.TemplateRoleDeleted,
			},
		}

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // Testing ACL validation
		err := client.UpdateTemplateACL(ctx, template.ID, req)
		require.Error(t, err)
		cerr, _ := nicloudsdk.AsError(err)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	t.Run("InvalidUser", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				uuid.NewString(): "admin",
			},
		}

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // Testing ACL validation
		err := client.UpdateTemplateACL(ctx, template.ID, req)
		require.Error(t, err)
		cerr, _ := nicloudsdk.AsError(err)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	// We should allow the special "Delete" role for valid UUIDs that don't
	// correspond to a valid user, because the user might have been deleted.
	t.Run("DeleteRoleForDeletedUser", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		_, deletedUser := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		//nolint:gocritic // Can't delete yourself
		err := client.DeleteUser(ctx, deletedUser.ID)
		require.NoError(t, err)

		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				deletedUser.ID.String(): nicloudsdk.TemplateRoleDeleted,
			},
		}
		//nolint:gocritic // Testing ACL validation
		err = client.UpdateTemplateACL(ctx, template.ID, req)
		require.NoError(t, err)
	})

	t.Run("DeletedUser", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		_, deletedUser := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		//nolint:gocritic // Can't delete yourself
		err := client.DeleteUser(ctx, deletedUser.ID)
		require.NoError(t, err)

		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				deletedUser.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		}
		//nolint:gocritic // Testing ACL validation
		err = client.UpdateTemplateACL(ctx, template.ID, req)
		require.Error(t, err)
		cerr, _ := nicloudsdk.AsError(err)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	t.Run("InvalidRole", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		_, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): "updater",
			},
		}

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // Testing ACL validation
		err := client.UpdateTemplateACL(ctx, template.ID, req)
		require.Error(t, err)
		cerr, _ := nicloudsdk.AsError(err)
		require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
	})

	t.Run("RegularUserCannotUpdatePerms", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		client1, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		client2, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): nicloudsdk.TemplateRoleUse,
			},
		}

		ctx := testutil.Context(t, testutil.WaitLong)

		err := client1.UpdateTemplateACL(ctx, template.ID, req)
		require.NoError(t, err)

		req = nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		}

		err = client2.UpdateTemplateACL(ctx, template.ID, req)
		require.Error(t, err)
		cerr, _ := nicloudsdk.AsError(err)
		require.Equal(t, http.StatusInternalServerError, cerr.StatusCode())
	})

	t.Run("RegularUserWithAdminCanUpdate", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		client1, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		client2, user2 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		req := nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user2.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		}

		// Group adds complexity to the /available endpoint
		// Intentionally omit user2
		nicloudtest.CreateGroup(t, client, user.OrganizationID, "some-group", user3)

		ctx := testutil.Context(t, testutil.WaitLong)

		err := client1.UpdateTemplateACL(ctx, template.ID, req)
		require.NoError(t, err)

		// Should be able to see user 3
		available, err := client2.TemplateACLAvailable(ctx, template.ID, nicloudsdk.UsersRequest{})
		require.NoError(t, err)
		userFound := false
		for _, avail := range available.Users {
			if avail.ID == user3.ID {
				userFound = true
			}
		}
		require.True(t, userFound, "user not found in acl available")

		req = nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				user3.ID.String(): nicloudsdk.TemplateRoleUse,
			},
		}

		err = client2.UpdateTemplateACL(ctx, template.ID, req)
		require.NoError(t, err)

		acl, err := client2.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		found := false
		for _, u := range acl.Users {
			if u.ID == user3.ID {
				found = true
			}
		}
		require.True(t, found, "user not found in acl")
	})

	t.Run("allUsersGroup", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Len(t, acl.Groups, 1)
		require.Len(t, acl.Users, 0)
	})

	t.Run("CustomGroupHasAccess", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin(), rbac.RoleUserAdmin())

		client1, user1 := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		// Create a group to add to the template.
		group, err := anotherClient.CreateGroup(ctx, user.OrganizationID, nicloudsdk.CreateGroupRequest{
			Name: "test",
		})
		require.NoError(t, err)

		// Check that the only current group is the allUsers group.
		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)
		require.Len(t, acl.Groups, 1)

		// Update the template to only allow access to the 'test' group.
		err = anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			GroupPerms: map[string]nicloudsdk.TemplateRole{
				// The allUsers group shares the same ID as the organization.
				user.OrganizationID.String(): nicloudsdk.TemplateRoleDeleted,
				group.ID.String():            nicloudsdk.TemplateRoleUse,
			},
		})
		require.NoError(t, err)

		// Get the ACL list for the template and assert the test group is
		// present.
		acl, err = anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Len(t, acl.Groups, 1)
		require.Len(t, acl.Users, 0)
		require.Equal(t, group.ID, acl.Groups[0].ID)

		// Try to get the template as the regular user. This should
		// fail since we haven't been added to the template yet.
		_, err = client1.Template(ctx, template.ID)
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, cerr.StatusCode())

		// Patch the group to add the regular user.
		group, err = anotherClient.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{user1.ID.String()},
		})
		require.NoError(t, err)
		require.Len(t, group.Members, 1)
		require.Equal(t, user1.ID, group.Members[0].ID)

		// Fetching the template should succeed since our group has view access.
		_, err = client1.Template(ctx, template.ID)
		require.NoError(t, err)
	})

	t.Run("NoAccess", func(t *testing.T) {
		t.Parallel()
		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleTemplateAdmin())

		client1, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		acl, err := anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Len(t, acl.Groups, 1)
		require.Len(t, acl.Users, 0)

		// User should be able to read template due to allUsers group.
		_, err = client1.Template(ctx, template.ID)
		require.NoError(t, err)

		allUsers := acl.Groups[0]

		err = anotherClient.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			GroupPerms: map[string]nicloudsdk.TemplateRole{
				allUsers.ID.String(): nicloudsdk.TemplateRoleDeleted,
			},
		})
		require.NoError(t, err)

		acl, err = anotherClient.TemplateACL(ctx, template.ID)
		require.NoError(t, err)

		require.Len(t, acl.Groups, 0)
		require.Len(t, acl.Users, 0)

		// User should not be able to read template due to allUsers group being deleted.
		_, err = client1.Template(ctx, template.ID)
		require.Error(t, err)
		cerr, ok := nicloudsdk.AsError(err)
		require.True(t, ok)
		require.Equal(t, http.StatusNotFound, cerr.StatusCode())
	})
}

func TestReadFileWithTemplateUpdate(t *testing.T) {
	t.Parallel()
	t.Run("HasTemplateUpdate", func(t *testing.T) {
		t.Parallel()

		// Upload a file
		client, first := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		ctx := testutil.Context(t, testutil.WaitLong)

		//nolint:gocritic // regular user cannot create file
		resp, err := client.Upload(ctx, nicloudsdk.ContentTypeTar, bytes.NewReader(make([]byte, 1024)))
		require.NoError(t, err)

		// Make a new user
		member, memberData := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID)

		// Try to download file, this should fail
		_, _, err = member.Download(ctx, resp.ID)
		require.Error(t, err, "no template yet")

		// Make a new template version with the file
		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil, func(request *nicloudsdk.CreateTemplateVersionRequest) {
			request.FileID = resp.ID
		})
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)

		// Not in acl yet
		_, _, err = member.Download(ctx, resp.ID)
		require.Error(t, err, "not in acl yet")

		//nolint:gocritic // regular user cannot update template acl
		err = client.UpdateTemplateACL(ctx, template.ID, nicloudsdk.UpdateTemplateACL{
			UserPerms: map[string]nicloudsdk.TemplateRole{
				memberData.ID.String(): nicloudsdk.TemplateRoleAdmin,
			},
		})
		require.NoError(t, err)

		_, _, err = member.Download(ctx, resp.ID)
		require.NoError(t, err)
	})
}

// TestTemplateAccess tests the rego -> sql conversion. We need to implement
// this test on at least 1 table type to ensure that the conversion is correct.
// The rbac tests only assert against static SQL queries.
// This is a full rbac test of many of the common role combinations.
//
//nolint:tparallel
func TestTemplateAccess(t *testing.T) {
	t.Parallel()
	// TODO: This context is for all the subtests. Each subtest should have its
	// own context.
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong*3)
	t.Cleanup(cancel)

	dv := nicloudtest.DeploymentValues(t)
	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			DeploymentValues: dv,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC:          1,
				nicloudsdk.FeatureMultipleOrganizations: 1,
			},
		},
	})

	type coderUser struct {
		*nicloudsdk.Client
		User nicloudsdk.User
	}

	type orgSetup struct {
		Admin         coderUser
		MemberInGroup coderUser
		MemberNoGroup coderUser

		DefaultTemplate nicloudsdk.Template
		AllRead         nicloudsdk.Template
		UserACL         nicloudsdk.Template
		GroupACL        nicloudsdk.Template

		Group nicloudsdk.Group
		Org   nicloudsdk.Organization
	}

	// Create the following users
	// - owner: Site wide owner
	// - template-admin
	// - org-admin (org 1)
	// - org-admin (org 2)
	// - org-member (org 1)
	// - org-member (org 2)

	// Create the following templates in each org
	// - template 1, default acls
	// - template 2, all_user read
	// - template 3, user_acl read for member
	// - template 4, group_acl read for groupMember

	templateAdmin, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())

	makeTemplate := func(t *testing.T, client *nicloudsdk.Client, orgID uuid.UUID, acl nicloudsdk.UpdateTemplateACL) nicloudsdk.Template {
		version := nicloudtest.CreateTemplateVersion(t, client, orgID, nil)
		template := nicloudtest.CreateTemplate(t, client, orgID, version.ID)

		err := client.UpdateTemplateACL(ctx, template.ID, acl)
		require.NoError(t, err, "failed to update template acl")

		return template
	}

	makeOrg := func(t *testing.T) orgSetup {
		// Make org
		orgName, err := cryptorand.String(5)
		require.NoError(t, err, "org name")

		// Make users
		newOrg, err := ownerClient.CreateOrganization(ctx, nicloudsdk.CreateOrganizationRequest{Name: orgName})
		require.NoError(t, err, "failed to create org")

		adminCli, adminUsr := nicloudtest.CreateAnotherUser(t, ownerClient, newOrg.ID, rbac.ScopedRoleOrgAdmin(newOrg.ID))
		groupMemCli, groupMemUsr := nicloudtest.CreateAnotherUser(t, ownerClient, newOrg.ID, rbac.ScopedRoleOrgMember(newOrg.ID))
		memberCli, memberUsr := nicloudtest.CreateAnotherUser(t, ownerClient, newOrg.ID, rbac.ScopedRoleOrgMember(newOrg.ID))

		// Make group
		group, err := adminCli.CreateGroup(ctx, newOrg.ID, nicloudsdk.CreateGroupRequest{
			Name: "SingleUser",
		})
		require.NoError(t, err, "failed to create group")

		group, err = adminCli.PatchGroup(ctx, group.ID, nicloudsdk.PatchGroupRequest{
			AddUsers: []string{groupMemUsr.ID.String()},
		})
		require.NoError(t, err, "failed to add user to group")

		// Make templates

		return orgSetup{
			Admin:         coderUser{Client: adminCli, User: adminUsr},
			MemberInGroup: coderUser{Client: groupMemCli, User: groupMemUsr},
			MemberNoGroup: coderUser{Client: memberCli, User: memberUsr},
			Org:           newOrg,
			Group:         group,

			DefaultTemplate: makeTemplate(t, adminCli, newOrg.ID, nicloudsdk.UpdateTemplateACL{
				GroupPerms: map[string]nicloudsdk.TemplateRole{
					newOrg.ID.String(): nicloudsdk.TemplateRoleDeleted,
				},
			}),
			AllRead: makeTemplate(t, adminCli, newOrg.ID, nicloudsdk.UpdateTemplateACL{
				GroupPerms: map[string]nicloudsdk.TemplateRole{
					newOrg.ID.String(): nicloudsdk.TemplateRoleUse,
				},
			}),
			UserACL: makeTemplate(t, adminCli, newOrg.ID, nicloudsdk.UpdateTemplateACL{
				GroupPerms: map[string]nicloudsdk.TemplateRole{
					newOrg.ID.String(): nicloudsdk.TemplateRoleDeleted,
				},
				UserPerms: map[string]nicloudsdk.TemplateRole{
					memberUsr.ID.String(): nicloudsdk.TemplateRoleUse,
				},
			}),
			GroupACL: makeTemplate(t, adminCli, newOrg.ID, nicloudsdk.UpdateTemplateACL{
				GroupPerms: map[string]nicloudsdk.TemplateRole{
					group.ID.String():  nicloudsdk.TemplateRoleUse,
					newOrg.ID.String(): nicloudsdk.TemplateRoleDeleted,
				},
			}),
		}
	}

	// Make 2 organizations
	orgs := []orgSetup{
		makeOrg(t),
		makeOrg(t),
	}

	testTemplateRead := func(t *testing.T, org orgSetup, usr *nicloudsdk.Client, read []nicloudsdk.Template) {
		found, err := usr.TemplatesByOrganization(ctx, org.Org.ID)
		if len(read) == 0 && err != nil {
			require.ErrorContains(t, err, "Resource not found")
			return
		}
		require.NoError(t, err, "failed to get templates")

		exp := make(map[uuid.UUID]nicloudsdk.Template)
		for _, tmpl := range read {
			exp[tmpl.ID] = tmpl
		}

		for _, f := range found {
			if _, ok := exp[f.ID]; !ok {
				t.Errorf("found unexpected template %q", f.Name)
			}
			delete(exp, f.ID)
		}
		require.Len(t, exp, 0, "expected templates not found")
	}

	// nolint:paralleltest
	t.Run("OwnerReadAll", func(t *testing.T) {
		for _, o := range orgs {
			// Owners can read all templates in all orgs
			exp := []nicloudsdk.Template{o.DefaultTemplate, o.AllRead, o.UserACL, o.GroupACL}
			testTemplateRead(t, o, ownerClient, exp)
		}
	})

	// nolint:paralleltest
	t.Run("TemplateAdminReadAll", func(t *testing.T) {
		for _, o := range orgs {
			// Template Admins can read all templates in all orgs
			exp := []nicloudsdk.Template{o.DefaultTemplate, o.AllRead, o.UserACL, o.GroupACL}
			testTemplateRead(t, o, templateAdmin, exp)
		}
	})

	// nolint:paralleltest
	t.Run("OrgAdminReadAllTheirs", func(t *testing.T) {
		for i, o := range orgs {
			cli := o.Admin.Client
			// Only read their own org
			exp := []nicloudsdk.Template{o.DefaultTemplate, o.AllRead, o.UserACL, o.GroupACL}
			testTemplateRead(t, o, cli, exp)

			other := orgs[(i+1)%len(orgs)]
			require.NotEqual(t, other.Org.ID, o.Org.ID, "this test needs at least 2 orgs")
			testTemplateRead(t, other, cli, []nicloudsdk.Template{})
		}
	})

	// nolint:paralleltest
	t.Run("TestMemberNoGroup", func(t *testing.T) {
		for i, o := range orgs {
			cli := o.MemberNoGroup.Client
			// Only read their own org
			exp := []nicloudsdk.Template{o.AllRead, o.UserACL}
			testTemplateRead(t, o, cli, exp)

			other := orgs[(i+1)%len(orgs)]
			require.NotEqual(t, other.Org.ID, o.Org.ID, "this test needs at least 2 orgs")
			testTemplateRead(t, other, cli, []nicloudsdk.Template{})
		}
	})

	// nolint:paralleltest
	t.Run("TestMemberInGroup", func(t *testing.T) {
		for i, o := range orgs {
			cli := o.MemberInGroup.Client
			// Only read their own org
			exp := []nicloudsdk.Template{o.AllRead, o.GroupACL}
			testTemplateRead(t, o, cli, exp)

			other := orgs[(i+1)%len(orgs)]
			require.NotEqual(t, other.Org.ID, o.Org.ID, "this test needs at least 2 orgs")
			testTemplateRead(t, other, cli, []nicloudsdk.Template{})
		}
	})
}

func TestMultipleOrganizationTemplates(t *testing.T) {
	t.Parallel()

	dv := nicloudtest.DeploymentValues(t)
	ownerClient, first := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			// This only affects the first org.
			IncludeProvisionerDaemon: true,
			DeploymentValues:         dv,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureExternalProvisionerDaemons: 1,
				nicloudsdk.FeatureMultipleOrganizations:      1,
			},
		},
	})

	templateAdmin, _ := nicloudtest.CreateAnotherUser(t, ownerClient, first.OrganizationID, rbac.RoleTemplateAdmin())

	second := nicloudenttest.CreateOrganization(t, ownerClient, nicloudenttest.CreateOrganizationOptions{
		IncludeProvisionerDaemon: true,
	})

	third := nicloudenttest.CreateOrganization(t, ownerClient, nicloudenttest.CreateOrganizationOptions{
		IncludeProvisionerDaemon: true,
	})

	t.Logf("First organization: %s", first.OrganizationID.String())
	t.Logf("Second organization: %s", second.ID.String())
	t.Logf("Third organization: %s", third.ID.String())

	t.Log("Creating template version in second organization")

	start := time.Now()
	version := nicloudtest.CreateTemplateVersion(t, templateAdmin, second.ID, nil)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, ownerClient, version.ID)
	nicloudtest.CreateTemplate(t, templateAdmin, second.ID, version.ID, func(request *nicloudsdk.CreateTemplateRequest) {
		request.Name = "random"
	})

	if time.Since(start) > time.Second*10 {
		// The test can sometimes pass because 'AwaitTemplateVersionJobCompleted'
		// allows 25s, and the provisioner will check every 30s if not awakened
		// from the pubsub. So there is a chance it will pass. If it takes longer
		// than 10s, then it's a problem. The provisioner is not getting clearance.
		t.Error("Creating template version in second organization took too long")
		t.FailNow()
	}
}

func TestInvalidateTemplatePrebuilds(t *testing.T) {
	t.Parallel()

	// Given the following parameters and presets...
	templateVersionParameters := []*proto.RichParameter{
		{Name: "param1", Type: "string", Required: false, DefaultValue: "default1"},
		{Name: "param2", Type: "string", Required: false, DefaultValue: "default2"},
		{Name: "param3", Type: "string", Required: false, DefaultValue: "default3"},
	}
	presetWithParameters1 := &proto.Preset{
		Name: "Preset With Parameters 1",
		Parameters: []*proto.PresetParameter{
			{Name: "param1", Value: "value1"},
			{Name: "param2", Value: "value2"},
			{Name: "param3", Value: "value3"},
		},
	}
	presetWithParameters2 := &proto.Preset{
		Name: "Preset With Parameters 2",
		Parameters: []*proto.PresetParameter{
			{Name: "param1", Value: "value4"},
			{Name: "param2", Value: "value5"},
			{Name: "param3", Value: "value6"},
		},
	}

	presetWithParameters3 := &proto.Preset{
		Name: "Preset With Parameters 3",
		Parameters: []*proto.PresetParameter{
			{Name: "param1", Value: "value7"},
			{Name: "param2", Value: "value8"},
			{Name: "param3", Value: "value9"},
		},
	}

	// Given the template versions and template...
	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureWorkspacePrebuilds: 1,
			},
		},
	})
	templateAdminClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())

	buildGraphResponse := func(presets ...*proto.Preset) *proto.Response {
		return &proto.Response{
			Type: &proto.Response_Graph{
				Graph: &proto.GraphComplete{
					Presets:    presets,
					Parameters: templateVersionParameters,
				},
			},
		}
	}

	version1 := nicloudtest.CreateTemplateVersion(t, templateAdminClient, owner.OrganizationID, &echo.Responses{
		Parse:          echo.ParseComplete,
		ProvisionApply: echo.ApplyComplete,
		ProvisionGraph: []*proto.Response{buildGraphResponse(presetWithParameters1, presetWithParameters2)},
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, version1.ID)
	template := nicloudtest.CreateTemplate(t, templateAdminClient, owner.OrganizationID, version1.ID)

	// When
	ctx := testutil.Context(t, testutil.WaitLong)
	invalidated, err := templateAdminClient.InvalidateTemplatePresets(ctx, template.ID)
	require.NoError(t, err)

	// Then
	require.Len(t, invalidated.Invalidated, 2)
	require.Equal(t, nicloudsdk.InvalidatedPreset{TemplateName: template.Name, TemplateVersionName: version1.Name, PresetName: presetWithParameters1.Name}, invalidated.Invalidated[0])
	require.Equal(t, nicloudsdk.InvalidatedPreset{TemplateName: template.Name, TemplateVersionName: version1.Name, PresetName: presetWithParameters2.Name}, invalidated.Invalidated[1])

	// Given the template is updated...
	version2 := nicloudtest.UpdateTemplateVersion(t, templateAdminClient, owner.OrganizationID, &echo.Responses{
		Parse:          echo.ParseComplete,
		ProvisionGraph: []*proto.Response{buildGraphResponse(presetWithParameters2, presetWithParameters3)},
		ProvisionApply: echo.ApplyComplete,
	}, template.ID)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, version2.ID)
	err = templateAdminClient.UpdateActiveTemplateVersion(ctx, template.ID, nicloudsdk.UpdateActiveTemplateVersion{ID: version2.ID})
	require.NoError(t, err)

	// When
	invalidated, err = templateAdminClient.InvalidateTemplatePresets(ctx, template.ID)
	require.NoError(t, err)

	// Then: it should only invalidate the presets from the currently active version (preset2 and preset3)
	require.Len(t, invalidated.Invalidated, 2)
	require.Equal(t, nicloudsdk.InvalidatedPreset{TemplateName: template.Name, TemplateVersionName: version2.Name, PresetName: presetWithParameters2.Name}, invalidated.Invalidated[0])
	require.Equal(t, nicloudsdk.InvalidatedPreset{TemplateName: template.Name, TemplateVersionName: version2.Name, PresetName: presetWithParameters3.Name}, invalidated.Invalidated[1])
}

func TestInvalidateTemplatePrebuilds_RegularUser(t *testing.T) {
	t.Parallel()

	// Given the following parameters and presets...
	templateVersionParameters := []*proto.RichParameter{
		{Name: "param1", Type: "string", Required: false, DefaultValue: "default1"},
	}
	presetWithParameters1 := &proto.Preset{
		Name: "Preset With Parameters 1",
		Parameters: []*proto.PresetParameter{
			{Name: "param1", Value: "value1"},
		},
	}

	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureWorkspacePrebuilds: 1,
			},
		},
	})
	regularUserClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

	// Given
	version1 := nicloudtest.CreateTemplateVersion(t, ownerClient, owner.OrganizationID, &echo.Responses{
		Parse: echo.ParseComplete,
		ProvisionGraph: []*proto.Response{
			{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Presets:    []*proto.Preset{presetWithParameters1},
						Parameters: templateVersionParameters,
					},
				},
			},
		},
		ProvisionApply: echo.ApplyComplete,
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, ownerClient, version1.ID)
	template := nicloudtest.CreateTemplate(t, ownerClient, owner.OrganizationID, version1.ID)

	// When
	ctx := testutil.Context(t, testutil.WaitShort)
	_, err := regularUserClient.InvalidateTemplatePresets(ctx, template.ID)

	// Then
	require.Error(t, err, "regular user cannot invalidate presets")
	var sdkError *nicloudsdk.Error
	require.True(t, errors.As(err, &sdkError))
	require.ErrorAs(t, err, &sdkError)
	require.Equal(t, http.StatusNotFound, sdkError.StatusCode())
}

func TestInvalidateTemplatePrebuilds_NoPresets(t *testing.T) {
	t.Parallel()

	// Given the template versions and template...
	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureWorkspacePrebuilds: 1,
			},
		},
	})
	templateAdminClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())

	version1 := nicloudtest.CreateTemplateVersion(t, templateAdminClient, owner.OrganizationID, &echo.Responses{
		Parse: echo.ParseComplete, ProvisionPlan: echo.PlanComplete, ProvisionApply: echo.ApplyComplete,
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, version1.ID)
	template := nicloudtest.CreateTemplate(t, templateAdminClient, owner.OrganizationID, version1.ID)

	// When
	ctx := testutil.Context(t, testutil.WaitLong)
	invalidated, err := templateAdminClient.InvalidateTemplatePresets(ctx, template.ID)
	require.NoError(t, err)

	// Then
	require.NotNil(t, invalidated.Invalidated)
	require.Len(t, invalidated.Invalidated, 0)
}

func TestInvalidateTemplatePrebuilds_LicenseFeatureDisabled(t *testing.T) {
	t.Parallel()

	// Given the template versions and template...
	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		},
		LicenseOptions: &nicloudenttest.LicenseOptions{},
	})
	templateAdminClient, _ := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())

	version1 := nicloudtest.CreateTemplateVersion(t, templateAdminClient, owner.OrganizationID, &echo.Responses{
		Parse: echo.ParseComplete, ProvisionPlan: echo.PlanComplete, ProvisionApply: echo.ApplyComplete,
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, version1.ID)
	template := nicloudtest.CreateTemplate(t, templateAdminClient, owner.OrganizationID, version1.ID)

	// When
	ctx := testutil.Context(t, testutil.WaitLong)
	_, err := templateAdminClient.InvalidateTemplatePresets(ctx, template.ID)

	// Then
	require.Error(t, err, "license feature prebuilds is required")
	var sdkError *nicloudsdk.Error
	require.True(t, errors.As(err, &sdkError))
	require.ErrorAs(t, err, &sdkError)
	require.Equal(t, http.StatusForbidden, sdkError.StatusCode())
}
