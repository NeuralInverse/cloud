package nicloud_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/xerrors"

	"cdr.dev/slog/v3"
	"cdr.dev/slog/v3/sloggers/slogtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest/oidctest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/db2sdk"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbgen"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtestutil"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtime"
	"github.com/NeuralInverse/cloud/v2/nicloud/externalauth"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications/notificationstest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestWorkspaceBuild(t *testing.T) {
	t.Parallel()
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	ctx := testutil.Context(t, testutil.WaitLong)
	auditor := audit.NewMock()
	client, db := nicloudtest.NewWithDatabase(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: true,
		Auditor:                  auditor,
	})
	user := nicloudtest.CreateFirstUser(t, client)
	up, err := db.UpdateUserProfile(dbauthz.AsSystemRestricted(ctx), database.UpdateUserProfileParams{
		ID:        user.UserID,
		Email:     nicloudtest.FirstUserParams.Email,
		Username:  nicloudtest.FirstUserParams.Username,
		Name:      "Admin",
		AvatarURL: client.URL.String(),
		UpdatedAt: dbtime.Now(),
	})
	require.NoError(t, err)
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
	template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	auditor.ResetLogs()
	workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
	_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
	// Create workspace will also start a build, so we need to wait for
	// it to ensure all events are recorded.
	require.Eventually(t, func() bool {
		logs := auditor.AuditLogs()
		return len(logs) == 2 &&
			assert.Equal(t, logs[0].Ip.IPNet.IP.String(), "127.0.0.1") &&
			assert.Equal(t, logs[1].Ip.IPNet.IP.String(), "127.0.0.1")
	}, testutil.WaitShort, testutil.IntervalFast)
	wb, err := client.WorkspaceBuild(testutil.Context(t, testutil.WaitShort), workspace.LatestBuild.ID)
	require.NoError(t, err)
	require.Equal(t, up.Username, wb.WorkspaceOwnerName)
	require.Equal(t, up.AvatarURL, wb.WorkspaceOwnerAvatarURL)
}

func TestWorkspaceBuildByBuildNumber(t *testing.T) {
	t.Parallel()
	t.Run("Successful", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		first := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		user, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err, "fetch me")
		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		_, err = client.WorkspaceBuildByUsernameAndWorkspaceNameAndBuildNumber(
			ctx,
			user.Username,
			workspace.Name,
			strconv.FormatInt(int64(workspace.LatestBuild.BuildNumber), 10),
		)
		require.NoError(t, err)
	})

	t.Run("BuildNumberNotInt", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		first := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		user, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err, "fetch me")
		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		_, err = client.WorkspaceBuildByUsernameAndWorkspaceNameAndBuildNumber(
			ctx,
			user.Username,
			workspace.Name,
			"buildNumber",
		)
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusBadRequest, apiError.StatusCode())
		require.ErrorContains(t, apiError, "Failed to parse build number as integer.")
	})

	t.Run("WorkspaceNotFound", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		first := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		user, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err, "fetch me")
		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		_, err = client.WorkspaceBuildByUsernameAndWorkspaceNameAndBuildNumber(
			ctx,
			user.Username,
			"workspaceName",
			strconv.FormatInt(int64(workspace.LatestBuild.BuildNumber), 10),
		)
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusNotFound, apiError.StatusCode())
		require.ErrorContains(t, apiError, "Resource not found")
	})

	t.Run("WorkspaceBuildNotFound", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		first := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		user, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err, "fetch me")
		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		_, err = client.WorkspaceBuildByUsernameAndWorkspaceNameAndBuildNumber(
			ctx,
			user.Username,
			workspace.Name,
			"200",
		)
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusNotFound, apiError.StatusCode())
		require.ErrorContains(t, apiError, fmt.Sprintf("Workspace %q Build 200 does not exist.", workspace.Name))
	})
}

func TestWorkspaceBuilds(t *testing.T) {
	t.Parallel()
	t.Run("Single", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		first := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		user, err := client.User(ctx, nicloudsdk.Me)
		require.NoError(t, err, "fetch me")
		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		builds, err := client.WorkspaceBuilds(ctx,
			nicloudsdk.WorkspaceBuildsRequest{WorkspaceID: workspace.ID})
		require.Len(t, builds, 1)
		require.Equal(t, int32(1), builds[0].BuildNumber)
		require.Equal(t, user.Username, builds[0].InitiatorUsername)
		require.NoError(t, err)

		// Test since
		builds, err = client.WorkspaceBuilds(ctx,
			nicloudsdk.WorkspaceBuildsRequest{WorkspaceID: workspace.ID, Since: dbtime.Now().Add(time.Minute)},
		)
		require.NoError(t, err)
		require.Len(t, builds, 0)
		// Should never be nil for API consistency
		require.NotNil(t, builds)

		builds, err = client.WorkspaceBuilds(ctx,
			nicloudsdk.WorkspaceBuildsRequest{WorkspaceID: workspace.ID, Since: dbtime.Now().Add(-time.Hour)},
		)
		require.NoError(t, err)
		require.Len(t, builds, 1)
	})

	t.Run("DeletedInitiator", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		first := nicloudtest.CreateFirstUser(t, client)
		second, secondUser := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleOwner())

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace, err := second.CreateWorkspace(ctx, first.OrganizationID, first.UserID.String(), nicloudsdk.CreateWorkspaceRequest{
			TemplateID: template.ID,
			Name:       "example",
		})
		require.NoError(t, err)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		err = client.DeleteUser(ctx, secondUser.ID)
		require.NoError(t, err)

		builds, err := client.WorkspaceBuilds(ctx, nicloudsdk.WorkspaceBuildsRequest{WorkspaceID: workspace.ID})
		require.Len(t, builds, 1)
		require.Equal(t, int32(1), builds[0].BuildNumber)
		require.Equal(t, secondUser.Username, builds[0].InitiatorUsername)
		require.NoError(t, err)
	})

	t.Run("PaginateNonExistentRow", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.WorkspaceBuilds(ctx, nicloudsdk.WorkspaceBuildsRequest{
			WorkspaceID: workspace.ID,
			Pagination: nicloudsdk.Pagination{
				AfterID: uuid.New(),
			},
		})
		var apiError *nicloudsdk.Error
		require.ErrorAs(t, err, &apiError)
		require.Equal(t, http.StatusBadRequest, apiError.StatusCode())
		require.Contains(t, apiError.Message, "does not exist")
	})

	t.Run("PaginateLimitOffset", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		var expectedBuilds []nicloudsdk.WorkspaceBuild
		extraBuilds := 4
		for i := 0; i < extraBuilds; i++ {
			b := nicloudtest.CreateWorkspaceBuild(t, client, workspace, database.WorkspaceTransitionStart)
			expectedBuilds = append(expectedBuilds, b)
			nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, b.ID)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		pageSize := 3
		firstPage, err := client.WorkspaceBuilds(ctx, nicloudsdk.WorkspaceBuildsRequest{
			WorkspaceID: workspace.ID,
			Pagination:  nicloudsdk.Pagination{Limit: pageSize, Offset: 0},
		})
		require.NoError(t, err)
		require.Len(t, firstPage, pageSize)
		for i := 0; i < pageSize; i++ {
			require.Equal(t, expectedBuilds[extraBuilds-i-1].ID, firstPage[i].ID)
		}
		secondPage, err := client.WorkspaceBuilds(ctx, nicloudsdk.WorkspaceBuildsRequest{
			WorkspaceID: workspace.ID,
			Pagination:  nicloudsdk.Pagination{Limit: pageSize, Offset: pageSize},
		})
		require.NoError(t, err)
		require.Len(t, secondPage, 2)
		require.Equal(t, expectedBuilds[0].ID, secondPage[0].ID)
		require.Equal(t, workspace.LatestBuild.ID, secondPage[1].ID) // build created while creating workspace
	})
}

func TestWorkspaceBuildsProvisionerState(t *testing.T) {
	t.Parallel()

	t.Run("Permissions", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		first := nicloudtest.CreateFirstUser(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: workspace.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionDelete,
			ProvisionerState:  []byte(" "),
		})
		require.Nil(t, err)

		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, build.ID)

		// A regular user on the very same template must not be able to modify the
		// state.
		regularUser, _ := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID)

		workspace = nicloudtest.CreateWorkspace(t, regularUser, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, regularUser, workspace.LatestBuild.ID)

		_, err = regularUser.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: workspace.LatestBuild.TemplateVersionID,
			Transition:        workspace.LatestBuild.Transition,
			ProvisionerState:  []byte(" "),
		})
		require.Error(t, err)

		var cerr *nicloudsdk.Error
		require.True(t, errors.As(err, &cerr))

		code := cerr.StatusCode()
		require.Equal(t, http.StatusForbidden, code, "unexpected status %s", http.StatusText(code))
	})

	t.Run("Orphan", func(t *testing.T) {
		t.Parallel()

		t.Run("WithoutDelete", func(t *testing.T) {
			t.Parallel()
			client, store := nicloudtest.NewWithDatabase(t, nil)
			first := nicloudtest.CreateFirstUser(t, client)
			templateAdmin, templateAdminUser := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleTemplateAdmin())

			r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
				OwnerID:        templateAdminUser.ID,
				OrganizationID: first.OrganizationID,
			}).Do()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			// Trying to orphan without delete transition fails.
			_, err := templateAdmin.CreateWorkspaceBuild(ctx, r.Workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
				TemplateVersionID: r.TemplateVersion.ID,
				Transition:        nicloudsdk.WorkspaceTransitionStart,
				Orphan:            true,
			})
			require.Error(t, err, "Orphan is only permitted when deleting a workspace.")
			cerr := nicloudtest.SDKError(t, err)
			require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
		})

		t.Run("WithState", func(t *testing.T) {
			t.Parallel()
			client, store := nicloudtest.NewWithDatabase(t, nil)
			first := nicloudtest.CreateFirstUser(t, client)
			templateAdmin, templateAdminUser := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleTemplateAdmin())

			r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
				OwnerID:        templateAdminUser.ID,
				OrganizationID: first.OrganizationID,
			}).Do()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			// Providing both state and orphan fails.
			_, err := templateAdmin.CreateWorkspaceBuild(ctx, r.Workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
				TemplateVersionID: r.TemplateVersion.ID,
				Transition:        nicloudsdk.WorkspaceTransitionDelete,
				ProvisionerState:  []byte(" "),
				Orphan:            true,
			})
			require.Error(t, err)
			cerr := nicloudtest.SDKError(t, err)
			require.Equal(t, http.StatusBadRequest, cerr.StatusCode())
		})

		t.Run("NoPermission", func(t *testing.T) {
			t.Parallel()
			client, store := nicloudtest.NewWithDatabase(t, nil)
			first := nicloudtest.CreateFirstUser(t, client)
			member, memberUser := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID)

			r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
				OwnerID:        memberUser.ID,
				OrganizationID: first.OrganizationID,
			}).Do()

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()

			// Trying to orphan without being a template admin fails.
			_, err := member.CreateWorkspaceBuild(ctx, r.Workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
				TemplateVersionID: r.TemplateVersion.ID,
				Transition:        nicloudsdk.WorkspaceTransitionDelete,
				Orphan:            true,
			})
			require.Error(t, err)
			cerr := nicloudtest.SDKError(t, err)
			require.Equal(t, http.StatusForbidden, cerr.StatusCode())
		})

		t.Run("OK", func(t *testing.T) {
			// Include a provisioner so that we can test that provisionerdserver
			// performs deletion.
			auditor := audit.NewMock()
			client, store := nicloudtest.NewWithDatabase(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, Auditor: auditor})
			first := nicloudtest.CreateFirstUser(t, client)
			templateAdmin, templateAdminUser := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleTemplateAdmin())

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()
			// This is a valid zip file. Without this the job will fail to complete.
			// TODO: add this to dbfake by default.
			zipBytes := make([]byte, 22)
			zipBytes[0] = 80
			zipBytes[1] = 75
			zipBytes[2] = 0o5
			zipBytes[3] = 0o6
			uploadRes, err := client.Upload(ctx, nicloudsdk.ContentTypeZip, bytes.NewReader(zipBytes))
			require.NoError(t, err)

			tv := dbfake.TemplateVersion(t, store).
				FileID(uploadRes.ID).
				Seed(database.TemplateVersion{
					OrganizationID: first.OrganizationID,
					CreatedBy:      templateAdminUser.ID,
				}).
				Do()

			r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
				OwnerID:        templateAdminUser.ID,
				OrganizationID: first.OrganizationID,
				TemplateID:     tv.Template.ID,
			}).Do()

			auditor.ResetLogs()
			// Regular orphan operation succeeds.
			build, err := templateAdmin.CreateWorkspaceBuild(ctx, r.Workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
				TemplateVersionID: r.TemplateVersion.ID,
				Transition:        nicloudsdk.WorkspaceTransitionDelete,
				Orphan:            true,
			})
			require.NoError(t, err)
			nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, build.ID)

			// Validate that the deletion was audited. This happens after the transaction
			// is committed, so it may not show up in the mock auditor immediately.
			testutil.Eventually(ctx, t, func(context.Context) bool {
				return auditor.Contains(t, database.AuditLog{
					ResourceID: build.ID,
					Action:     database.AuditActionDelete,
				})
			}, testutil.IntervalFast)
		})

		t.Run("NoProvisioners", func(t *testing.T) {
			t.Parallel()
			auditor := audit.NewMock()
			client, store := nicloudtest.NewWithDatabase(t, &nicloudtest.Options{Auditor: auditor})
			first := nicloudtest.CreateFirstUser(t, client)
			templateAdmin, templateAdminUser := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleTemplateAdmin())

			ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
			defer cancel()
			r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
				OwnerID:        templateAdminUser.ID,
				OrganizationID: first.OrganizationID,
			}).Do()

			daemons, err := store.GetProvisionerDaemons(dbauthz.AsSystemReadProvisionerDaemons(ctx))
			require.NoError(t, err)
			require.Empty(t, daemons, "Provisioner daemons should be empty for this test")

			// Orphan deletion still succeeds despite no provisioners being available.
			build, err := templateAdmin.CreateWorkspaceBuild(ctx, r.Workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
				TemplateVersionID: r.TemplateVersion.ID,
				Transition:        nicloudsdk.WorkspaceTransitionDelete,
				Orphan:            true,
			})
			require.NoError(t, err)
			require.Equal(t, nicloudsdk.WorkspaceTransitionDelete, build.Transition)
			require.Equal(t, nicloudsdk.ProvisionerJobSucceeded, build.Job.Status)
			require.Empty(t, build.Job.Error)

			ws, err := client.Workspace(ctx, r.Workspace.ID)
			require.Empty(t, ws)
			require.Equal(t, http.StatusGone, nicloudtest.SDKError(t, err).StatusCode())

			// Validate that the deletion was audited. This happens after the transaction
			// is committed, so it may not show up in the mock auditor immediately.
			testutil.Eventually(ctx, t, func(context.Context) bool {
				return auditor.Contains(t, database.AuditLog{
					ResourceID: build.ID,
					Action:     database.AuditActionDelete,
				})
			}, testutil.IntervalFast)
		})
	})
}

func TestPatchCancelWorkspaceBuild(t *testing.T) {
	t.Parallel()
	t.Run("User is allowed to cancel", func(t *testing.T) {
		t.Parallel()

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionInit:  echo.InitComplete,
			ProvisionGraph: echo.GraphComplete,
			ProvisionPlan:  echo.PlanComplete,
			// Echo will never applying since there is no complete message
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{},
				},
			}},
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		var build nicloudsdk.WorkspaceBuild

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		require.Eventually(t, func() bool {
			var err error
			build, err = client.WorkspaceBuild(ctx, workspace.LatestBuild.ID)
			return assert.NoError(t, err) && build.Job.Status == nicloudsdk.ProvisionerJobRunning
		}, testutil.WaitShort, testutil.IntervalFast)

		require.Eventually(t, func() bool {
			err := client.CancelWorkspaceBuild(ctx, build.ID, nicloudsdk.CancelWorkspaceBuildParams{})
			return err == nil
		}, testutil.WaitShort, testutil.IntervalMedium)

		require.Eventually(t, func() bool {
			var err error
			build, err = client.WorkspaceBuild(ctx, build.ID)
			// job gets marked Failed when there is an Error; in practice we never get to Status = Canceled
			// because provisioners report an Error when canceled. We check the Error string to ensure we don't mask
			// other errors in this test.
			return assert.NoError(t, err) &&
				build.Job.Error == "canceled" &&
				build.Job.Status == nicloudsdk.ProvisionerJobFailed
		}, testutil.WaitShort, testutil.IntervalFast)
	})
	t.Run("User is not allowed to cancel", func(t *testing.T) {
		t.Parallel()

		// need to include our own logger because the provisioner (rightly) drops error logs when we shut down the
		// test with a build in progress.
		logger := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true}).Leveled(slog.LevelDebug)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, Logger: &logger})
		owner := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionInit:  echo.InitComplete,
			ProvisionGraph: echo.GraphComplete,
			ProvisionPlan:  echo.PlanComplete,
			// Echo will never applying
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{},
				},
			}},
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)

		userClient, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		workspace := nicloudtest.CreateWorkspace(t, userClient, template.ID)
		var build nicloudsdk.WorkspaceBuild

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		require.Eventually(t, func() bool {
			var err error
			build, err = userClient.WorkspaceBuild(ctx, workspace.LatestBuild.ID)
			return assert.NoError(t, err) && build.Job.Status == nicloudsdk.ProvisionerJobRunning
		}, testutil.WaitShort, testutil.IntervalFast)
		err := userClient.CancelWorkspaceBuild(ctx, build.ID, nicloudsdk.CancelWorkspaceBuildParams{})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusForbidden, apiErr.StatusCode())
	})

	t.Run("Cancel with expect_state=pending", func(t *testing.T) {
		t.Parallel()

		// Given: a nicloud instance with a provisioner daemon
		store, ps, db := dbtestutil.NewDBWithSQLDB(t)
		client, closeDaemon := nicloudtest.NewWithProvisionerCloser(t, &nicloudtest.Options{
			Database:                 store,
			Pubsub:                   ps,
			IncludeProvisionerDaemon: true,
		})
		defer closeDaemon.Close()
		// Given: a user, template, and workspace
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		// Stop the provisioner daemon.
		require.NoError(t, closeDaemon.Close())
		ctx := testutil.Context(t, testutil.WaitLong)
		// Given: no provisioner daemons exist.
		_, err := db.ExecContext(ctx, `DELETE FROM provisioner_daemons;`)
		require.NoError(t, err)

		// When: a new workspace build is created
		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		// Then: the request should succeed.
		require.NoError(t, err)
		// Then: the provisioner job should remain pending.
		require.Equal(t, nicloudsdk.ProvisionerJobPending, build.Job.Status)

		// Then: the response should indicate no provisioners are available.
		if assert.NotNil(t, build.MatchedProvisioners) {
			assert.Zero(t, build.MatchedProvisioners.Count)
			assert.Zero(t, build.MatchedProvisioners.Available)
			assert.Zero(t, build.MatchedProvisioners.MostRecentlySeen.Time)
			assert.False(t, build.MatchedProvisioners.MostRecentlySeen.Valid)
		}

		// When: the workspace build is canceled
		err = client.CancelWorkspaceBuild(ctx, build.ID, nicloudsdk.CancelWorkspaceBuildParams{
			ExpectStatus: nicloudsdk.CancelWorkspaceBuildStatusPending,
		})
		require.NoError(t, err)

		// Then: the workspace build should be canceled.
		build, err = client.WorkspaceBuild(ctx, build.ID)
		require.NoError(t, err)
		require.Equal(t, nicloudsdk.ProvisionerJobCanceled, build.Job.Status)
	})

	t.Run("Cancel with expect_state=pending when job is running - should fail with 412", func(t *testing.T) {
		t.Parallel()

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionInit:  echo.InitComplete,
			ProvisionGraph: echo.GraphComplete,
			ProvisionPlan:  echo.PlanComplete,
			// Echo will never applying
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{},
				},
			}},
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		var build nicloudsdk.WorkspaceBuild
		require.Eventually(t, func() bool {
			var err error
			build, err = client.WorkspaceBuild(ctx, workspace.LatestBuild.ID)
			return assert.NoError(t, err) && build.Job.Status == nicloudsdk.ProvisionerJobRunning
		}, testutil.WaitShort, testutil.IntervalFast)

		// When: a cancel request is made with expect_state=pending
		err := client.CancelWorkspaceBuild(ctx, build.ID, nicloudsdk.CancelWorkspaceBuildParams{
			ExpectStatus: nicloudsdk.CancelWorkspaceBuildStatusPending,
		})
		// Then: the request should fail with 412.
		require.Error(t, err)

		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusPreconditionFailed, apiErr.StatusCode())
	})

	t.Run("Cancel with expect_state=running when job is pending - should fail with 412", func(t *testing.T) {
		t.Parallel()

		// Given: a nicloud instance with a provisioner daemon
		store, ps, db := dbtestutil.NewDBWithSQLDB(t)
		client, closeDaemon := nicloudtest.NewWithProvisionerCloser(t, &nicloudtest.Options{
			Database:                 store,
			Pubsub:                   ps,
			IncludeProvisionerDaemon: true,
		})
		defer closeDaemon.Close()
		// Given: a user, template, and workspace
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		// Stop the provisioner daemon.
		require.NoError(t, closeDaemon.Close())
		ctx := testutil.Context(t, testutil.WaitLong)
		// Given: no provisioner daemons exist.
		_, err := db.ExecContext(ctx, `DELETE FROM provisioner_daemons;`)
		require.NoError(t, err)

		// When: a new workspace build is created
		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		// Then: the request should succeed.
		require.NoError(t, err)
		// Then: the provisioner job should remain pending.
		require.Equal(t, nicloudsdk.ProvisionerJobPending, build.Job.Status)

		// Then: the response should indicate no provisioners are available.
		if assert.NotNil(t, build.MatchedProvisioners) {
			assert.Zero(t, build.MatchedProvisioners.Count)
			assert.Zero(t, build.MatchedProvisioners.Available)
			assert.Zero(t, build.MatchedProvisioners.MostRecentlySeen.Time)
			assert.False(t, build.MatchedProvisioners.MostRecentlySeen.Valid)
		}

		// When: a cancel request is made with expect_state=running
		err = client.CancelWorkspaceBuild(ctx, build.ID, nicloudsdk.CancelWorkspaceBuildParams{
			ExpectStatus: nicloudsdk.CancelWorkspaceBuildStatusRunning,
		})
		// Then: the request should fail with 412.
		require.Error(t, err)

		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusPreconditionFailed, apiErr.StatusCode())
	})

	t.Run("Cancel with expect_state - invalid status", func(t *testing.T) {
		t.Parallel()

		// Given: a nicloud instance with a provisioner daemon
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionInit:  echo.InitComplete,
			ProvisionGraph: echo.GraphComplete,
			ProvisionPlan:  echo.PlanComplete,
			// Echo will never applying
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{},
				},
			}},
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)

		ctx := testutil.Context(t, testutil.WaitLong)

		// When: a cancel request is made with invalid expect_state
		err := client.CancelWorkspaceBuild(ctx, workspace.LatestBuild.ID, nicloudsdk.CancelWorkspaceBuildParams{
			ExpectStatus: "invalid_status",
		})
		// Then: the request should fail with 400.
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
		require.Contains(t, apiErr.Message, "Invalid expect_status")
	})
}

func TestWorkspaceBuildResources(t *testing.T) {
	t.Parallel()
	t.Run("List", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionGraph: []*proto.Response{{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Resources: []*proto.Resource{{
							Name: "first_resource",
							Type: "example",
							Agents: []*proto.Agent{{
								Id:    "something-1",
								Name:  "something-1",
								Auth:  &proto.Agent_Token{},
								Order: 3,
							}},
						}, {
							Name: "second_resource",
							Type: "example",
							Agents: []*proto.Agent{{
								Id:    "something-2",
								Name:  "something-2",
								Auth:  &proto.Agent_Token{},
								Order: 1,
							}, {
								Id:    "something-3",
								Name:  "something-3",
								Auth:  &proto.Agent_Token{},
								Order: 2,
							}},
						}, {
							Name: "third_resource",
							Type: "example",
						}, {
							Name: "fourth_resource",
							Type: "example",
						}, {
							Name: "fifth_resource",
							Type: "example",
							Agents: []*proto.Agent{{
								Id:   "something-4",
								Name: "something-4",
								Auth: &proto.Agent_Token{},
							}},
						}},
					},
				},
			}},
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		workspace, err := client.Workspace(ctx, workspace.ID)
		require.NoError(t, err)
		require.NotNil(t, workspace.LatestBuild.Resources)
		require.Len(t, workspace.LatestBuild.Resources, 5)
		assertWorkspaceResource(t, workspace.LatestBuild.Resources[0], "fifth_resource", "example", 1)  // resource has agent with implicit order = 0
		assertWorkspaceResource(t, workspace.LatestBuild.Resources[1], "second_resource", "example", 2) // resource has 2 agents, one with low order value (2)
		assertWorkspaceResource(t, workspace.LatestBuild.Resources[2], "first_resource", "example", 1)  // resource has 1 agent with explicit order
		assertWorkspaceResource(t, workspace.LatestBuild.Resources[3], "fourth_resource", "example", 0) // resource has no agents, sorted by name
		assertWorkspaceResource(t, workspace.LatestBuild.Resources[4], "third_resource", "example", 0)  // resource is the last one
	})
}

func TestWorkspaceBuildWithUpdatedTemplateVersionSendsNotification(t *testing.T) {
	t.Parallel()

	t.Run("NoRepeatedNotifications", func(t *testing.T) {
		t.Parallel()

		notify := &notificationstest.FakeEnqueuer{}

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, NotificationsEnqueuer: notify})
		first := nicloudtest.CreateFirstUser(t, client)
		templateAdminClient, templateAdmin := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleTemplateAdmin())
		userClient, user := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID)

		// Create a template with an initial version
		version := nicloudtest.CreateTemplateVersion(t, templateAdminClient, first.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, version.ID)
		template := nicloudtest.CreateTemplate(t, templateAdminClient, first.OrganizationID, version.ID)

		// Create a workspace using this template
		workspace := nicloudtest.CreateWorkspace(t, userClient, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, userClient, workspace.LatestBuild.ID)
		nicloudtest.MustTransitionWorkspace(t, userClient, workspace.ID, nicloudsdk.WorkspaceTransitionStart, nicloudsdk.WorkspaceTransitionStop)

		// Create a new version of the template
		newVersion := nicloudtest.CreateTemplateVersion(t, templateAdminClient, first.OrganizationID, nil, func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
			ctvr.TemplateID = template.ID
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, newVersion.ID)

		// Create a workspace build using this new template version
		build := nicloudtest.CreateWorkspaceBuild(t, userClient, workspace, database.WorkspaceTransitionStart, func(cwbr *nicloudsdk.CreateWorkspaceBuildRequest) {
			cwbr.TemplateVersionID = newVersion.ID
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, userClient, build.ID)
		nicloudtest.MustTransitionWorkspace(t, userClient, workspace.ID, nicloudsdk.WorkspaceTransitionStart, nicloudsdk.WorkspaceTransitionStop)

		// Create the workspace build _again_. We are doing this to
		// ensure we do not create _another_ notification. This is
		// separate to the notifications subsystem dedupe mechanism
		// as this build shouldn't create a notification. It shouldn't
		// create another notification as this new build isn't changing
		// the template version.
		build = nicloudtest.CreateWorkspaceBuild(t, userClient, workspace, database.WorkspaceTransitionStart, func(cwbr *nicloudsdk.CreateWorkspaceBuildRequest) {
			cwbr.TemplateVersionID = newVersion.ID
		})
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, userClient, build.ID)
		nicloudtest.MustTransitionWorkspace(t, userClient, workspace.ID, nicloudsdk.WorkspaceTransitionStart, nicloudsdk.WorkspaceTransitionStop)

		// We're going to have two notifications (one for the first user and one for the template admin)
		// By ensuring we only have these two, we are sure the second build didn't trigger more
		// notifications.
		sent := notify.Sent(notificationstest.WithTemplateID(notifications.TemplateWorkspaceManuallyUpdated))
		require.Len(t, sent, 2)

		receivers := make([]uuid.UUID, len(sent))
		for idx, notif := range sent {
			receivers[idx] = notif.UserID
		}

		// Check the notification was sent to the first user and template admin
		// (both of whom have the "template admin" role), and explicitly not the
		// workspace owner (since they initiated the workspace build).
		require.Contains(t, receivers, templateAdmin.ID)
		require.Contains(t, receivers, first.UserID)
		require.NotContains(t, receivers, user.ID)

		require.Contains(t, sent[0].Targets, template.ID)
		require.Contains(t, sent[0].Targets, workspace.ID)
		require.Contains(t, sent[0].Targets, workspace.OrganizationID)
		require.Contains(t, sent[0].Targets, workspace.OwnerID)

		require.Contains(t, sent[1].Targets, template.ID)
		require.Contains(t, sent[1].Targets, workspace.ID)
		require.Contains(t, sent[1].Targets, workspace.OrganizationID)
		require.Contains(t, sent[1].Targets, workspace.OwnerID)
	})

	t.Run("ToCorrectUser", func(t *testing.T) {
		t.Parallel()

		notify := &notificationstest.FakeEnqueuer{}

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, NotificationsEnqueuer: notify})
		first := nicloudtest.CreateFirstUser(t, client)
		templateAdminClient, templateAdmin := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID, rbac.RoleTemplateAdmin())
		userClient, user := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID)

		// Create a template with an initial version
		version := nicloudtest.CreateTemplateVersion(t, templateAdminClient, first.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, version.ID)
		template := nicloudtest.CreateTemplate(t, templateAdminClient, first.OrganizationID, version.ID)

		// Create a workspace using this template
		workspace := nicloudtest.CreateWorkspace(t, userClient, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, userClient, workspace.LatestBuild.ID)
		nicloudtest.MustTransitionWorkspace(t, userClient, workspace.ID, nicloudsdk.WorkspaceTransitionStart, nicloudsdk.WorkspaceTransitionStop)

		// Create a new version of the template
		newVersion := nicloudtest.CreateTemplateVersion(t, templateAdminClient, first.OrganizationID, nil, func(ctvr *nicloudsdk.CreateTemplateVersionRequest) {
			ctvr.TemplateID = template.ID
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAdminClient, newVersion.ID)

		// Create a workspace build using this new template version from a different user
		ctx := testutil.Context(t, testutil.WaitShort)
		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			Transition:        nicloudsdk.WorkspaceTransitionStart,
			TemplateVersionID: newVersion.ID,
		})
		require.NoError(t, err)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, userClient, build.ID)
		nicloudtest.MustTransitionWorkspace(t, userClient, workspace.ID, nicloudsdk.WorkspaceTransitionStart, nicloudsdk.WorkspaceTransitionStop)

		// Ensure we receive only 1 workspace manually updated notification and to the right user
		sent := notify.Sent(notificationstest.WithTemplateID(notifications.TemplateWorkspaceManuallyUpdated))
		require.Len(t, sent, 1)
		require.Equal(t, templateAdmin.ID, sent[0].UserID)
		require.Contains(t, sent[0].Targets, template.ID)
		require.Contains(t, sent[0].Targets, workspace.ID)
		require.Contains(t, sent[0].Targets, workspace.OrganizationID)
		require.Contains(t, sent[0].Targets, workspace.OwnerID)

		owner, ok := sent[0].Data["owner"].(map[string]any)
		require.True(t, ok, "notification data should have owner")
		require.Equal(t, user.ID, owner["id"])
		require.Equal(t, user.Name, owner["name"])
		require.Equal(t, user.Email, owner["email"])
	})
}

func assertWorkspaceResource(t *testing.T, actual nicloudsdk.WorkspaceResource, name, aType string, numAgents int) {
	assert.Equal(t, name, actual.Name)
	assert.Equal(t, aType, actual.Type)
	assert.Len(t, actual.Agents, numAgents)
}

func TestWorkspaceBuildLogs(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
	user := nicloudtest.CreateFirstUser(t, client)
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse: echo.ParseComplete,
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
							Name: "dev",
							Auth: &proto.Agent_Token{},
						}},
					}, {
						Name: "another",
						Type: "example",
					}},
				},
			},
		}},
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
	workspace := nicloudtest.CreateWorkspace(t, client, template.ID)

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	logs, closer, err := client.WorkspaceBuildLogsAfter(ctx, workspace.LatestBuild.ID, 0)
	require.NoError(t, err)
	defer closer.Close()
	for {
		log, ok := <-logs
		if !ok {
			break
		}
		if log.Output == "example" {
			return
		}
	}
	require.Fail(t, "example message never happened")
}

func TestWorkspaceBuildLogsFormat(t *testing.T) {
	t.Parallel()

	// Setup: Create workspace build with logs using dbfake.
	client, db := nicloudtest.NewWithDatabase(t, nil)
	user := nicloudtest.CreateFirstUser(t, client)

	r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
		OrganizationID: user.OrganizationID,
		OwnerID:        user.UserID,
	}).Do()

	// Insert test log directly into database.
	jl := dbgen.ProvisionerJobLog(t, db, database.ProvisionerJobLog{
		JobID:  r.Build.JobID,
		Stage:  "Planning",
		Source: database.LogSourceProvisioner,
		Level:  database.LogLevelInfo,
		Output: "test log output",
	})

	tests := []struct {
		name                string
		queryParams         string
		expectedStatus      int
		expectedContentType string
		checkBody           func(t *testing.T, body string)
	}{
		{
			name:                "JSON",
			queryParams:         "",
			expectedStatus:      http.StatusOK,
			expectedContentType: "application/json",
			checkBody: func(t *testing.T, body string) {
				require.NotEmpty(t, body)
			},
		},
		{
			name:                "Text",
			queryParams:         "?format=text",
			expectedStatus:      http.StatusOK,
			expectedContentType: "text/plain",
			checkBody: func(t *testing.T, body string) {
				expected := db2sdk.ProvisionerJobLog(jl).Text()
				require.Contains(t, body, expected)
			},
		},
		{
			name:           "InvalidFormat",
			queryParams:    "?format=invalid",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				require.Contains(t, body, "Invalid format")
			},
		},
		{
			name:           "TextWithFollowFails",
			queryParams:    "?format=text&follow",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				require.Contains(t, body, "not supported with follow mode")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := testutil.Context(t, testutil.WaitLong)

			urlPath := fmt.Sprintf("/api/v2/workspacebuilds/%s/logs%s", r.Build.ID, tt.queryParams)

			res, err := client.Request(ctx, http.MethodGet, urlPath, nil)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, tt.expectedStatus, res.StatusCode)
			if tt.expectedContentType != "" {
				require.Contains(t, res.Header.Get("Content-Type"), tt.expectedContentType)
			}

			if assert.NotNil(t, tt.checkBody) {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				tt.checkBody(t, string(body))
			}
		})
	}
}

func TestWorkspaceBuildState(t *testing.T) {
	t.Parallel()
	client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
	user := nicloudtest.CreateFirstUser(t, client)
	wantState := []byte("some kinda state")
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
		Parse:         echo.ParseComplete,
		ProvisionPlan: echo.PlanComplete,
		ProvisionApply: []*proto.Response{{
			Type: &proto.Response_Apply{
				Apply: &proto.ApplyComplete{
					State: wantState,
				},
			},
		}},
	})
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
	workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
	nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	gotState, err := client.WorkspaceBuildState(ctx, workspace.LatestBuild.ID)
	require.NoError(t, err)
	require.Equal(t, wantState, gotState)
}

func TestWorkspaceBuildStatus(t *testing.T) {
	t.Parallel()

	auditor := audit.NewMock()
	numLogs := len(auditor.AuditLogs())
	client, closeDaemon, api := nicloudtest.NewWithAPI(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, Auditor: auditor})
	user := nicloudtest.CreateFirstUser(t, client)
	numLogs++ // add an audit log for login
	version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
	numLogs++ // add an audit log for template version creation
	numLogs++ // add an audit log for template version update

	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	closeDaemon.Close()
	template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
	numLogs++ // add an audit log for template creation

	workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
	numLogs++ // add an audit log for workspace creation

	// initial returned state is "pending"
	require.EqualValues(t, nicloudsdk.WorkspaceStatusPending, workspace.LatestBuild.Status)

	closeDaemon = nicloudtest.NewProvisionerDaemon(t, api)
	// after successful build is "running"
	_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	workspace, err := client.Workspace(ctx, workspace.ID)
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceStatusRunning, workspace.LatestBuild.Status)

	numLogs++ // add an audit log for workspace_build starting

	// after successful stop is "stopped"
	build := nicloudtest.CreateWorkspaceBuild(t, client, workspace, database.WorkspaceTransitionStop)
	_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, build.ID)
	workspace, err = client.Workspace(ctx, workspace.ID)
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceStatusStopped, workspace.LatestBuild.Status)

	// assert an audit log has been created for workspace stopping
	numLogs++ // add an audit log for workspace_build stop
	require.Len(t, auditor.AuditLogs(), numLogs)
	require.Equal(t, database.AuditActionStop, auditor.AuditLogs()[numLogs-1].Action)

	_ = closeDaemon.Close()
	// after successful cancel is "canceled"
	build = nicloudtest.CreateWorkspaceBuild(t, client, workspace, database.WorkspaceTransitionStart)
	err = client.CancelWorkspaceBuild(ctx, build.ID, nicloudsdk.CancelWorkspaceBuildParams{})
	require.NoError(t, err)

	workspace, err = client.Workspace(ctx, workspace.ID)
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceStatusCanceled, workspace.LatestBuild.Status)

	_ = nicloudtest.NewProvisionerDaemon(t, api)
	// after successful delete is "deleted"
	build = nicloudtest.CreateWorkspaceBuild(t, client, workspace, database.WorkspaceTransitionDelete)
	_ = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, build.ID)
	workspace, err = client.DeletedWorkspace(ctx, workspace.ID)
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceStatusDeleted, workspace.LatestBuild.Status)
}

func TestWorkspaceDeleteSuspendedUser(t *testing.T) {
	t.Parallel()
	const providerID = "fake-github"
	fake := oidctest.NewFakeIDP(t, oidctest.WithServing())

	validateCalls := 0
	userSuspended := false
	owner := nicloudtest.New(t, &nicloudtest.Options{
		IncludeProvisionerDaemon: true,
		ExternalAuthConfigs: []*externalauth.Config{
			fake.ExternalAuthConfig(t, providerID, &oidctest.ExternalAuthConfigOptions{
				ValidatePayload: func(email string) (interface{}, int, error) {
					validateCalls++
					if userSuspended {
						// Simulate the user being suspended from the IDP too.
						return "", http.StatusForbidden, xerrors.New("user is suspended")
					}
					return "OK", 0, nil
				},
			}),
		},
	})

	first := nicloudtest.CreateFirstUser(t, owner)

	// New user that we will suspend when we try to delete the workspace.
	client, user := nicloudtest.CreateAnotherUser(t, owner, first.OrganizationID, rbac.RoleTemplateAdmin())
	fake.ExternalLogin(t, client)

	version := nicloudtest.CreateTemplateVersion(t, client, first.OrganizationID, &echo.Responses{
		Parse:          echo.ParseComplete,
		ProvisionApply: echo.ApplyComplete,
		ProvisionGraph: []*proto.Response{{
			Type: &proto.Response_Graph{
				Graph: &proto.GraphComplete{
					Error:      "",
					Resources:  nil,
					Parameters: nil,
					ExternalAuthProviders: []*proto.ExternalAuthProviderResource{
						{
							Id:       providerID,
							Optional: false,
						},
					},
				},
			},
		}},
	})

	validateCalls = 0 // Reset
	nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
	template := nicloudtest.CreateTemplate(t, client, first.OrganizationID, version.ID)
	workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
	nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
	require.Equal(t, 1, validateCalls) // Ensure the external link is working

	// Suspend the user
	ctx := testutil.Context(t, testutil.WaitLong)
	_, err := owner.UpdateUserStatus(ctx, user.ID.String(), nicloudsdk.UserStatusSuspended)
	require.NoError(t, err, "suspend user")

	// Now delete the workspace build
	userSuspended = true
	build, err := owner.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
		Transition: nicloudsdk.WorkspaceTransitionDelete,
	})
	require.NoError(t, err)
	build = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, owner, build.ID)
	require.Equal(t, 2, validateCalls)
	require.Equal(t, nicloudsdk.WorkspaceStatusDeleted, build.Status)
}

func TestWorkspaceBuildDebugMode(t *testing.T) {
	t.Parallel()

	t.Run("DebugModeDisabled", func(t *testing.T) {
		t.Parallel()

		// Create user
		deploymentValues := nicloudtest.DeploymentValues(t)
		err := deploymentValues.EnableTerraformDebugMode.Set("false")
		require.NoError(t, err)

		adminClient := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, DeploymentValues: deploymentValues})
		owner := nicloudtest.CreateFirstUser(t, adminClient)

		// Template author: create a template
		version := nicloudtest.CreateTemplateVersion(t, adminClient, owner.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, adminClient, owner.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, adminClient, version.ID)

		// Template author: create a workspace
		workspace := nicloudtest.CreateWorkspace(t, adminClient, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, adminClient, workspace.LatestBuild.ID)

		// Template author: try to start a workspace build in debug mode
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err = adminClient.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: workspace.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
			LogLevel:          "debug",
		})

		// Template author: expect an error as the debug mode is disabled
		require.NotNil(t, err)
		var sdkError *nicloudsdk.Error
		isSdkError := xerrors.As(err, &sdkError)
		require.True(t, isSdkError)
		require.Contains(t, sdkError.Message, "Terraform debug mode is disabled in the deployment configuration.")
	})
	t.Run("AsRegularUser", func(t *testing.T) {
		t.Parallel()

		// Create users
		deploymentValues := nicloudtest.DeploymentValues(t)
		deploymentValues.EnableTerraformDebugMode = true

		templateAuthorClient := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, DeploymentValues: deploymentValues})
		templateAuthor := nicloudtest.CreateFirstUser(t, templateAuthorClient)
		regularUserClient, _ := nicloudtest.CreateAnotherUser(t, templateAuthorClient, templateAuthor.OrganizationID)

		// Template owner: create a template
		version := nicloudtest.CreateTemplateVersion(t, templateAuthorClient, templateAuthor.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, templateAuthorClient, templateAuthor.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAuthorClient, version.ID)

		// Regular user: create a workspace
		workspace := nicloudtest.CreateWorkspace(t, regularUserClient, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, regularUserClient, workspace.LatestBuild.ID)

		// Regular user: try to start a workspace build in debug mode
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := regularUserClient.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: workspace.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
			LogLevel:          "debug",
		})

		// Regular user: expect an error
		require.NotNil(t, err)
		var sdkError *nicloudsdk.Error
		isSdkError := xerrors.As(err, &sdkError)
		require.True(t, isSdkError)
		require.Contains(t, sdkError.Message, "Workspace builds with a custom log level are restricted to administrators only.")
	})
	t.Run("AsTemplateAuthor", func(t *testing.T) {
		t.Parallel()

		// Create users
		deploymentValues := nicloudtest.DeploymentValues(t)
		deploymentValues.EnableTerraformDebugMode = true

		adminClient := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, DeploymentValues: deploymentValues})
		owner := nicloudtest.CreateFirstUser(t, adminClient)
		templateAuthorClient, _ := nicloudtest.CreateAnotherUser(t, adminClient, owner.OrganizationID, rbac.RoleTemplateAdmin())

		// Template author: create a template
		version := nicloudtest.CreateTemplateVersion(t, templateAuthorClient, owner.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, templateAuthorClient, owner.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, templateAuthorClient, version.ID)

		// Template author: create a workspace
		workspace := nicloudtest.CreateWorkspace(t, templateAuthorClient, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, templateAuthorClient, workspace.LatestBuild.ID)

		// Template author: try to start a workspace build in debug mode
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := templateAuthorClient.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: workspace.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
			LogLevel:          "debug",
		})

		// Template author: expect an error as the debug mode is disabled
		require.NotNil(t, err)
		var sdkError *nicloudsdk.Error
		isSdkError := xerrors.As(err, &sdkError)
		require.True(t, isSdkError)
		require.Contains(t, sdkError.Message, "Workspace builds with a custom log level are restricted to administrators only.")
	})
	t.Run("AsAdmin", func(t *testing.T) {
		t.Parallel()

		// Create users
		deploymentValues := nicloudtest.DeploymentValues(t)
		deploymentValues.EnableTerraformDebugMode = true

		adminClient := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, DeploymentValues: deploymentValues})
		owner := nicloudtest.CreateFirstUser(t, adminClient)

		// Interact as template admin
		echoResponses := &echo.Responses{
			Parse:         echo.ParseComplete,
			ProvisionPlan: echo.PlanComplete,
			ProvisionApply: []*proto.Response{{
				Type: &proto.Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_DEBUG,
						Output: "want-it",
					},
				},
			}, {
				Type: &proto.Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_TRACE,
						Output: "dont-want-it",
					},
				},
			}, {
				Type: &proto.Response_Log{
					Log: &proto.Log{
						Level:  proto.LogLevel_DEBUG,
						Output: "done",
					},
				},
			}, {
				Type: &proto.Response_Apply{
					Apply: &proto.ApplyComplete{},
				},
			}},
		}
		version := nicloudtest.CreateTemplateVersion(t, adminClient, owner.OrganizationID, echoResponses)
		template := nicloudtest.CreateTemplate(t, adminClient, owner.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, adminClient, version.ID)

		// Create workspace
		workspace := nicloudtest.CreateWorkspace(t, adminClient, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, adminClient, workspace.LatestBuild.ID)

		// Create workspace build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		build, err := adminClient.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: workspace.LatestBuild.TemplateVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
			ProvisionerState:  []byte(" "),
			LogLevel:          "debug",
		})
		require.Nil(t, err)

		build = nicloudtest.AwaitWorkspaceBuildJobCompleted(t, adminClient, build.ID)

		// Watch for incoming logs
		logs, closer, err := adminClient.WorkspaceBuildLogsAfter(ctx, build.ID, 0)
		require.NoError(t, err)
		defer closer.Close()

		var logsProcessed int

	processingLogs:
		for {
			select {
			case <-ctx.Done():
				require.Fail(t, "timeout occurred while processing logs")
				return
			case log, ok := <-logs:
				if !ok {
					break processingLogs
				}
				t.Logf("got log: %s -- %s | %s | %s", log.Level, log.Stage, log.Source, log.Output)
				if log.Source != "provisioner" {
					continue
				}
				logsProcessed++

				require.NotEqual(t, "dont-want-it", log.Output, "unexpected log message", "%s log message shouldn't be logged: %s")

				if log.Output == "done" {
					break processingLogs
				}
			}
		}
		require.Equal(t, 2, logsProcessed)
	})
}

func TestPostWorkspaceBuild(t *testing.T) {
	t.Parallel()
	t.Run("NoTemplateVersion", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: uuid.New(),
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		require.Error(t, err)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})

	t.Run("TemplateVersionFailedImport", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			ProvisionPlan: []*proto.Response{
				{
					Type: &proto.Response_Plan{
						Plan: &proto.PlanComplete{
							Error: "failed to plan",
						},
					},
				},
			},
		})
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		version = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.CreateWorkspace(ctx, user.OrganizationID, nicloudsdk.Me, nicloudsdk.CreateWorkspaceRequest{
			TemplateID: template.ID,
			Name:       "workspace",
		})
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})

	t.Run("AlreadyActive", func(t *testing.T) {
		t.Parallel()
		client, closer := nicloudtest.NewWithProvisionerCloser(t, nil)
		defer closer.Close()

		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		closer.Close()
		// Close here so workspace build doesn't process!
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		_, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		require.Error(t, err)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.StatusCode())
	})

	t.Run("Audit", func(t *testing.T) {
		t.Parallel()

		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			),
		)
		auditor := audit.NewMock()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true, Auditor: auditor})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		auditor.ResetLogs()
		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		require.NoError(t, err)
		if assert.NotNil(t, build.MatchedProvisioners) {
			require.Equal(t, 1, build.MatchedProvisioners.Count)
			require.Equal(t, 1, build.MatchedProvisioners.Available)
			require.NotZero(t, build.MatchedProvisioners.MostRecentlySeen.Time)
		}

		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, build.ID)

		require.Eventually(t, func() bool {
			logs := auditor.AuditLogs()
			return len(logs) > 0 &&
				assert.Equal(t, logs[0].Ip.IPNet.IP.String(), "127.0.0.1")
		}, testutil.WaitShort, testutil.IntervalFast)
	})

	t.Run("IncrementBuildNumber", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		require.NoError(t, err)
		if assert.NotNil(t, build.MatchedProvisioners) {
			require.Equal(t, 1, build.MatchedProvisioners.Count)
			require.Equal(t, 1, build.MatchedProvisioners.Available)
			require.NotZero(t, build.MatchedProvisioners.MostRecentlySeen.Time)
		}

		require.Equal(t, workspace.LatestBuild.BuildNumber+1, build.BuildNumber)
	})

	t.Run("WithState", func(t *testing.T) {
		t.Parallel()
		client, closeDaemon := nicloudtest.NewWithProvisionerCloser(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		wantState := []byte("something")
		_ = closeDaemon.Close()

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
			ProvisionerState:  wantState,
		})
		require.NoError(t, err)
		if assert.NotNil(t, build.MatchedProvisioners) {
			require.Equal(t, 1, build.MatchedProvisioners.Count)
			require.Equal(t, 1, build.MatchedProvisioners.Available)
			require.NotZero(t, build.MatchedProvisioners.MostRecentlySeen.Time)
		}

		gotState, err := client.WorkspaceBuildState(ctx, build.ID)
		require.NoError(t, err)
		require.Equal(t, wantState, gotState)
	})

	t.Run("SetsPresetID", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitLong)

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, &echo.Responses{
			Parse: echo.ParseComplete,
			ProvisionGraph: []*proto.Response{{
				Type: &proto.Response_Graph{
					Graph: &proto.GraphComplete{
						Presets: []*proto.Preset{
							{
								Name: "autodetected",
							},
							{
								Name: "manual",
								Parameters: []*proto.PresetParameter{
									{
										Name:  "param1",
										Value: "value1",
									},
								},
							},
						},
					},
				},
			}},
			ProvisionApply: echo.ApplyComplete,
		})
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)

		presets, err := client.TemplateVersionPresets(ctx, version.ID)
		require.NoError(t, err)
		require.Equal(t, 2, len(presets))
		require.Equal(t, "autodetected", presets[0].Name)
		require.Equal(t, "manual", presets[1].Name)

		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		// Preset ID was detected based on the workspace parameters:
		require.Equal(t, presets[0].ID, *workspace.LatestBuild.TemplateVersionPresetID)

		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID:       version.ID,
			Transition:              nicloudsdk.WorkspaceTransitionStart,
			TemplateVersionPresetID: presets[1].ID,
		})
		require.NoError(t, err)
		require.NotNil(t, build.TemplateVersionPresetID)

		workspace, err = client.Workspace(ctx, workspace.ID)
		require.NoError(t, err)
		require.Equal(t, presets[1].ID, *workspace.LatestBuild.TemplateVersionPresetID)
		require.Equal(t, build.TemplateVersionPresetID, workspace.LatestBuild.TemplateVersionPresetID)
	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			Transition: nicloudsdk.WorkspaceTransitionDelete,
		})
		require.NoError(t, err)
		require.Equal(t, workspace.LatestBuild.BuildNumber+1, build.BuildNumber)
		if assert.NotNil(t, build.MatchedProvisioners) {
			require.Equal(t, 1, build.MatchedProvisioners.Count)
			require.Equal(t, 1, build.MatchedProvisioners.Available)
			require.NotZero(t, build.MatchedProvisioners.MostRecentlySeen.Time)
		}

		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, build.ID)

		res, err := client.Workspaces(ctx, nicloudsdk.WorkspaceFilter{
			Owner: user.UserID.String(),
		})
		require.NoError(t, err)
		require.Len(t, res.Workspaces, 0)
	})

	t.Run("NoProvisionersAvailable", func(t *testing.T) {
		t.Parallel()

		// Given: a nicloud instance with a provisioner daemon
		store, ps, db := dbtestutil.NewDBWithSQLDB(t)
		client, closeDaemon := nicloudtest.NewWithProvisionerCloser(t, &nicloudtest.Options{
			Database:                 store,
			Pubsub:                   ps,
			IncludeProvisionerDaemon: true,
		})
		defer closeDaemon.Close()
		// Given: a user, template, and workspace
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		// Stop the provisioner daemon.
		require.NoError(t, closeDaemon.Close())
		ctx := testutil.Context(t, testutil.WaitLong)
		// Given: no provisioner daemons exist.
		_, err := db.ExecContext(ctx, `DELETE FROM provisioner_daemons;`)
		require.NoError(t, err)

		// When: a new workspace build is created
		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		// Then: the request should succeed.
		require.NoError(t, err)
		// Then: the provisioner job should remain pending.
		require.Equal(t, nicloudsdk.ProvisionerJobPending, build.Job.Status)
		// Then: the response should indicate no provisioners are available.
		if assert.NotNil(t, build.MatchedProvisioners) {
			assert.Zero(t, build.MatchedProvisioners.Count)
			assert.Zero(t, build.MatchedProvisioners.Available)
			assert.Zero(t, build.MatchedProvisioners.MostRecentlySeen.Time)
			assert.False(t, build.MatchedProvisioners.MostRecentlySeen.Valid)
		}
	})

	t.Run("AllProvisionersStale", func(t *testing.T) {
		t.Parallel()

		// Given: a nicloud instance with a provisioner daemon
		store, ps, db := dbtestutil.NewDBWithSQLDB(t)
		client, closeDaemon := nicloudtest.NewWithProvisionerCloser(t, &nicloudtest.Options{
			Database:                 store,
			Pubsub:                   ps,
			IncludeProvisionerDaemon: true,
		})
		defer closeDaemon.Close()
		// Given: a user, template, and workspace
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)

		ctx := testutil.Context(t, testutil.WaitLong)
		// Given: all provisioner daemons are stale
		// First stop the provisioner
		require.NoError(t, closeDaemon.Close())
		newLastSeenAt := dbtime.Now().Add(-time.Hour)
		// Update the last seen at for all provisioner daemons. We have to use the
		// SQL db directly because store.UpdateProvisionerDaemonLastSeenAt has a
		// built-in check to prevent updating the last seen at to a time in the past.
		_, err := db.ExecContext(ctx, `UPDATE provisioner_daemons SET last_seen_at = $1;`, newLastSeenAt)
		require.NoError(t, err)

		// When: a new workspace build is created
		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
		})
		// Then: the request should succeed
		require.NoError(t, err)
		// Then: the provisioner job should remain pending
		require.Equal(t, nicloudsdk.ProvisionerJobPending, build.Job.Status)
		// Then: the response should indicate no provisioners are available
		if assert.NotNil(t, build.MatchedProvisioners) {
			assert.Zero(t, build.MatchedProvisioners.Available)
			assert.Equal(t, 1, build.MatchedProvisioners.Count)
			assert.Equal(t, newLastSeenAt.UTC(), build.MatchedProvisioners.MostRecentlySeen.Time.UTC())
			assert.True(t, build.MatchedProvisioners.MostRecentlySeen.Valid)
		}
	})
	t.Run("WithReason", func(t *testing.T) {
		t.Parallel()
		client, closeDaemon := nicloudtest.NewWithProvisionerCloser(t, &nicloudtest.Options{
			IncludeProvisionerDaemon: true,
		})
		user := nicloudtest.CreateFirstUser(t, client)
		version := nicloudtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := nicloudtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, client, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		_ = closeDaemon.Close()

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancel()

		build, err := client.CreateWorkspaceBuild(ctx, workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
			TemplateVersionID: template.ActiveVersionID,
			Transition:        nicloudsdk.WorkspaceTransitionStart,
			Reason:            nicloudsdk.CreateWorkspaceBuildReasonDashboard,
		})
		require.NoError(t, err)
		require.Equal(t, nicloudsdk.BuildReasonDashboard, build.Reason)
	})
	t.Run("DeletedWorkspace", func(t *testing.T) {
		t.Parallel()

		// Given: a workspace that has already been deleted
		var (
			ctx             = testutil.Context(t, testutil.WaitShort)
			logger          = slogtest.Make(t, &slogtest.Options{}).Leveled(slog.LevelError)
			adminClient, db = nicloudtest.NewWithDatabase(t, &nicloudtest.Options{
				Logger: &logger,
			})
			admin                         = nicloudtest.CreateFirstUser(t, adminClient)
			workspaceOwnerClient, member1 = nicloudtest.CreateAnotherUser(t, adminClient, admin.OrganizationID)
			otherMemberClient, _          = nicloudtest.CreateAnotherUser(t, adminClient, admin.OrganizationID)
			ws                            = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{OwnerID: member1.ID, OrganizationID: admin.OrganizationID}).
							Seed(database.WorkspaceBuild{Transition: database.WorkspaceTransitionDelete}).
							Do()
		)

		// This needs to be done separately as provisionerd handles marking the workspace as deleted
		// and we're skipping provisionerd here for speed.
		require.NoError(t, db.UpdateWorkspaceDeletedByID(dbauthz.AsProvisionerd(ctx), database.UpdateWorkspaceDeletedByIDParams{
			ID:      ws.Workspace.ID,
			Deleted: true,
		}))

		// Assert test invariant: Workspace should be deleted
		dbWs, err := db.GetWorkspaceByID(dbauthz.AsProvisionerd(ctx), ws.Workspace.ID)
		require.NoError(t, err)
		require.True(t, dbWs.Deleted, "workspace should be deleted")

		for _, tc := range []struct {
			user         *nicloudsdk.Client
			tr           nicloudsdk.WorkspaceTransition
			expectStatus int
		}{
			// You should not be allowed to mess with a workspace you don't own, regardless of its deleted state.
			{otherMemberClient, nicloudsdk.WorkspaceTransitionStart, http.StatusNotFound},
			{otherMemberClient, nicloudsdk.WorkspaceTransitionStop, http.StatusNotFound},
			{otherMemberClient, nicloudsdk.WorkspaceTransitionDelete, http.StatusNotFound},
			// Starting or stopping a workspace is not allowed when it is deleted.
			{workspaceOwnerClient, nicloudsdk.WorkspaceTransitionStart, http.StatusConflict},
			{workspaceOwnerClient, nicloudsdk.WorkspaceTransitionStop, http.StatusConflict},
			// We allow a delete just in case a retry is required. In most cases, this will be a no-op.
			// Note: this is the last test case because it will change the state of the workspace.
			{workspaceOwnerClient, nicloudsdk.WorkspaceTransitionDelete, http.StatusOK},
		} {
			// When: we create a workspace build with the given transition
			_, err = tc.user.CreateWorkspaceBuild(ctx, ws.Workspace.ID, nicloudsdk.CreateWorkspaceBuildRequest{
				Transition: tc.tr,
			})

			// Then: we allow ONLY a delete build for a deleted workspace.
			if tc.expectStatus < http.StatusBadRequest {
				require.NoError(t, err, "creating a %s build for a deleted workspace should not error", tc.tr)
			} else {
				var apiError *nicloudsdk.Error
				require.Error(t, err, "creating a %s build for a deleted workspace should return an error", tc.tr)
				require.ErrorAs(t, err, &apiError)
				require.Equal(t, tc.expectStatus, apiError.StatusCode())
			}
		}
	})
}

func TestWorkspaceBuildTimings(t *testing.T) {
	t.Parallel()

	// Setup the test environment with a template and version
	db, pubsub := dbtestutil.NewDB(t)
	ownerClient := nicloudtest.New(t, &nicloudtest.Options{
		Database: db,
		Pubsub:   pubsub,
	})
	owner := nicloudtest.CreateFirstUser(t, ownerClient)
	client, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

	file := dbgen.File(t, db, database.File{
		CreatedBy: owner.UserID,
	})
	versionJob := dbgen.ProvisionerJob(t, db, pubsub, database.ProvisionerJob{
		OrganizationID: owner.OrganizationID,
		InitiatorID:    user.ID,
		FileID:         file.ID,
		Tags: database.StringMap{
			"custom": "true",
		},
	})
	version := dbgen.TemplateVersion(t, db, database.TemplateVersion{
		OrganizationID: owner.OrganizationID,
		JobID:          versionJob.ID,
		CreatedBy:      owner.UserID,
	})
	template := dbgen.Template(t, db, database.Template{
		OrganizationID:  owner.OrganizationID,
		ActiveVersionID: version.ID,
		CreatedBy:       owner.UserID,
	})

	// Tests will run in parallel. To avoid conflicts and race conditions on the
	// build number, each test will have its own workspace and build.
	makeBuild := func(t *testing.T) database.WorkspaceBuild {
		ws := dbgen.Workspace(t, db, database.WorkspaceTable{
			OwnerID:        user.ID,
			OrganizationID: owner.OrganizationID,
			TemplateID:     template.ID,
		})
		jobID := uuid.New()
		job := dbgen.ProvisionerJob(t, db, pubsub, database.ProvisionerJob{
			ID:             jobID,
			OrganizationID: owner.OrganizationID,
			Tags:           database.StringMap{jobID.String(): "true"},
		})
		return dbgen.WorkspaceBuild(t, db, database.WorkspaceBuild{
			WorkspaceID:       ws.ID,
			TemplateVersionID: version.ID,
			InitiatorID:       owner.UserID,
			JobID:             job.ID,
			BuildNumber:       1,
		})
	}

	t.Run("NonExistentBuild", func(t *testing.T) {
		t.Parallel()

		// Given: a non-existent build
		buildID := uuid.New()

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		_, err := client.WorkspaceBuildTimings(ctx, buildID)

		// Then: expect a not found error
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("EmptyTimings", func(t *testing.T) {
		t.Parallel()

		// Given: a build with no timings
		build := makeBuild(t)

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)

		// Then: return a response with empty timings
		require.NoError(t, err)
		require.Empty(t, res.ProvisionerTimings)
		require.Empty(t, res.AgentScriptTimings)
	})

	t.Run("ProvisionerTimings", func(t *testing.T) {
		t.Parallel()

		// Given: a build with provisioner timings
		build := makeBuild(t)
		provisionerTimings := dbgen.ProvisionerJobTimings(t, db, build, 5)

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)
		require.NoError(t, err)

		// Then: return a response with the expected timings
		require.Len(t, res.ProvisionerTimings, 5)
		for i := range res.ProvisionerTimings {
			timingRes := res.ProvisionerTimings[i]
			genTiming := provisionerTimings[i]
			require.Equal(t, genTiming.Resource, timingRes.Resource)
			require.Equal(t, genTiming.Action, timingRes.Action)
			require.Equal(t, string(genTiming.Stage), string(timingRes.Stage))
			require.Equal(t, genTiming.JobID.String(), timingRes.JobID.String())
			require.Equal(t, genTiming.Source, timingRes.Source)
			require.Equal(t, genTiming.StartedAt.UnixMilli(), timingRes.StartedAt.UnixMilli())
			require.Equal(t, genTiming.EndedAt.UnixMilli(), timingRes.EndedAt.UnixMilli())
		}
	})

	t.Run("MultipleTimingsForSameAgentScript", func(t *testing.T) {
		t.Parallel()

		// Given: a build with multiple timings for the same script
		build := makeBuild(t)
		resource := dbgen.WorkspaceResource(t, db, database.WorkspaceResource{
			JobID: build.JobID,
		})
		agent := dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{
			ResourceID: resource.ID,
		})
		script := dbgen.WorkspaceAgentScript(t, db, database.WorkspaceAgentScript{
			WorkspaceAgentID: agent.ID,
		})
		timings := make([]database.WorkspaceAgentScriptTiming, 3)
		scriptStartedAt := dbtime.Now()
		for i := range timings {
			timings[i] = dbgen.WorkspaceAgentScriptTiming(t, db, database.WorkspaceAgentScriptTiming{
				StartedAt: scriptStartedAt,
				EndedAt:   scriptStartedAt.Add(1 * time.Minute),
				ScriptID:  script.ID,
			})

			// Add an hour to the previous "started at" so we can
			// reliably differentiate the scripts from each other.
			scriptStartedAt = scriptStartedAt.Add(1 * time.Hour)
		}

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)
		require.NoError(t, err)

		// Then: return a response with the first agent script timing
		require.Len(t, res.AgentScriptTimings, 1)

		require.Equal(t, timings[0].StartedAt.UnixMilli(), res.AgentScriptTimings[0].StartedAt.UnixMilli())
		require.Equal(t, timings[0].EndedAt.UnixMilli(), res.AgentScriptTimings[0].EndedAt.UnixMilli())
	})

	t.Run("AgentScriptTimings", func(t *testing.T) {
		t.Parallel()

		// Given: a build with agent script timings
		build := makeBuild(t)
		resource := dbgen.WorkspaceResource(t, db, database.WorkspaceResource{
			JobID: build.JobID,
		})
		agent := dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{
			ResourceID: resource.ID,
		})
		scripts := dbgen.WorkspaceAgentScripts(t, db, 5, database.WorkspaceAgentScript{
			WorkspaceAgentID: agent.ID,
		})
		agentScriptTimings := dbgen.WorkspaceAgentScriptTimings(t, db, scripts)

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)
		require.NoError(t, err)

		// Then: return a response with the expected timings
		require.Len(t, res.AgentScriptTimings, 5)
		slices.SortFunc(res.AgentScriptTimings, func(a, b nicloudsdk.AgentScriptTiming) int {
			return a.StartedAt.Compare(b.StartedAt)
		})
		slices.SortFunc(agentScriptTimings, func(a, b database.WorkspaceAgentScriptTiming) int {
			return a.StartedAt.Compare(b.StartedAt)
		})
		for i := range res.AgentScriptTimings {
			timingRes := res.AgentScriptTimings[i]
			genTiming := agentScriptTimings[i]
			require.Equal(t, genTiming.ExitCode, timingRes.ExitCode)
			require.Equal(t, string(genTiming.Status), timingRes.Status)
			require.Equal(t, string(genTiming.Stage), string(timingRes.Stage))
			require.Equal(t, genTiming.StartedAt.UnixMilli(), timingRes.StartedAt.UnixMilli())
			require.Equal(t, genTiming.EndedAt.UnixMilli(), timingRes.EndedAt.UnixMilli())
			require.Equal(t, agent.ID.String(), timingRes.WorkspaceAgentID)
			require.Equal(t, agent.Name, timingRes.WorkspaceAgentName)
		}
	})

	t.Run("NoAgentScripts", func(t *testing.T) {
		t.Parallel()

		// Given: a build with no agent scripts
		build := makeBuild(t)
		resource := dbgen.WorkspaceResource(t, db, database.WorkspaceResource{
			JobID: build.JobID,
		})
		dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{
			ResourceID: resource.ID,
		})

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)
		require.NoError(t, err)

		// Then: return a response with empty agent script timings
		require.Empty(t, res.AgentScriptTimings)
	})

	// Some workspaces might not have agents. It is improbable, but possible.
	t.Run("NoAgents", func(t *testing.T) {
		t.Parallel()

		// Given: a build with no agents
		build := makeBuild(t)
		dbgen.WorkspaceResource(t, db, database.WorkspaceResource{
			JobID: build.JobID,
		})

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)
		require.NoError(t, err)

		// Then: return a response with empty agent script timings
		require.Empty(t, res.AgentScriptTimings)
		require.Empty(t, res.AgentConnectionTimings)
	})

	t.Run("AgentConnectionTimings", func(t *testing.T) {
		t.Parallel()

		// Given: a build with an agent
		build := makeBuild(t)
		resource := dbgen.WorkspaceResource(t, db, database.WorkspaceResource{
			JobID: build.JobID,
		})
		agent := dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{
			ResourceID:       resource.ID,
			FirstConnectedAt: sql.NullTime{Valid: true, Time: dbtime.Now().Add(-time.Hour)},
		})

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)
		require.NoError(t, err)

		// Then: return a response with the expected timings
		require.Len(t, res.AgentConnectionTimings, 1)
		for i := range res.ProvisionerTimings {
			timingRes := res.AgentConnectionTimings[i]
			require.Equal(t, agent.ID.String(), timingRes.WorkspaceAgentID)
			require.Equal(t, agent.Name, timingRes.WorkspaceAgentName)
			require.NotEmpty(t, timingRes.StartedAt)
			require.NotEmpty(t, timingRes.EndedAt)
		}
	})

	t.Run("MultipleAgents", func(t *testing.T) {
		t.Parallel()

		// Given: a build with multiple agents
		build := makeBuild(t)
		resource := dbgen.WorkspaceResource(t, db, database.WorkspaceResource{
			JobID: build.JobID,
		})
		agents := make([]database.WorkspaceAgent, 5)
		for i := range agents {
			agents[i] = dbgen.WorkspaceAgent(t, db, database.WorkspaceAgent{
				ResourceID:       resource.ID,
				FirstConnectedAt: sql.NullTime{Valid: true, Time: dbtime.Now().Add(-time.Duration(i) * time.Hour)},
			})
		}

		// When: fetching timings for the build
		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
		t.Cleanup(cancel)
		res, err := client.WorkspaceBuildTimings(ctx, build.ID)
		require.NoError(t, err)

		// Then: return a response with the expected timings
		require.Len(t, res.AgentConnectionTimings, 5)
	})
}
