package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/provisioner/echo"
)

func TestStatePull(t *testing.T) {
	t.Parallel()
	t.Run("File", func(t *testing.T) {
		t.Parallel()
		client, store := nicloudtest.NewWithDatabase(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, taUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		wantState := []byte("some state")
		r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
			OrganizationID: owner.OrganizationID,
			OwnerID:        taUser.ID,
		}).
			Seed(database.WorkspaceBuild{}).ProvisionerState(wantState).
			Do()
		statefilePath := filepath.Join(t.TempDir(), "state")
		inv, root := clitest.New(t, "state", "pull", r.Workspace.Name, statefilePath)
		clitest.SetupConfig(t, templateAdmin, root)
		err := inv.Run()
		require.NoError(t, err)
		gotState, err := os.ReadFile(statefilePath)
		require.NoError(t, err)
		require.Equal(t, wantState, gotState)
	})
	t.Run("Stdout", func(t *testing.T) {
		t.Parallel()
		client, store := nicloudtest.NewWithDatabase(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, taUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		wantState := []byte("some state")
		r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
			OrganizationID: owner.OrganizationID,
			OwnerID:        taUser.ID,
		}).
			Seed(database.WorkspaceBuild{}).ProvisionerState(wantState).
			Do()
		inv, root := clitest.New(t, "state", "pull", r.Workspace.Name)
		var gotState bytes.Buffer
		inv.Stdout = &gotState
		clitest.SetupConfig(t, templateAdmin, root)
		err := inv.Run()
		require.NoError(t, err)
		require.Equal(t, wantState, bytes.TrimSpace(gotState.Bytes()))
	})
	t.Run("OtherUserBuild", func(t *testing.T) {
		t.Parallel()
		client, store := nicloudtest.NewWithDatabase(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		_, taUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		wantState := []byte("some state")
		r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
			OrganizationID: owner.OrganizationID,
			OwnerID:        taUser.ID,
		}).
			Seed(database.WorkspaceBuild{}).ProvisionerState(wantState).
			Do()
		inv, root := clitest.New(t, "state", "pull", taUser.Username+"/"+r.Workspace.Name,
			"--build", fmt.Sprintf("%d", r.Build.BuildNumber))
		var gotState bytes.Buffer
		inv.Stdout = &gotState
		//nolint: gocritic // this tests owner pulling another user's state
		clitest.SetupConfig(t, client, root)
		err := inv.Run()
		require.NoError(t, err)
		require.Equal(t, wantState, bytes.TrimSpace(gotState.Bytes()))
	})
}

func TestStatePush(t *testing.T) {
	t.Parallel()
	t.Run("File", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, templateAdmin, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		stateFile, err := os.CreateTemp(t.TempDir(), "")
		require.NoError(t, err)
		wantState := []byte("some magic state")
		_, err = stateFile.Write(wantState)
		require.NoError(t, err)
		err = stateFile.Close()
		require.NoError(t, err)
		inv, root := clitest.New(t, "state", "push", workspace.Name, stateFile.Name())
		clitest.SetupConfig(t, templateAdmin, root)
		err = inv.Run()
		require.NoError(t, err)
	})

	t.Run("Stdin", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, templateAdmin, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		inv, root := clitest.New(t, "state", "push", "--build", strconv.Itoa(int(workspace.LatestBuild.BuildNumber)), workspace.Name, "-")
		clitest.SetupConfig(t, templateAdmin, root)
		inv.Stdin = strings.NewReader("some magic state")
		err := inv.Run()
		require.NoError(t, err)
	})

	t.Run("OtherUserBuild", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, taUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, &echo.Responses{
			Parse:          echo.ParseComplete,
			ProvisionApply: echo.ApplyComplete,
		})
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, templateAdmin, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		inv, root := clitest.New(t, "state", "push",
			"--build", strconv.Itoa(int(workspace.LatestBuild.BuildNumber)),
			taUser.Username+"/"+workspace.Name,
			"-")
		//nolint: gocritic // this tests owner pushing another user's state
		clitest.SetupConfig(t, client, root)
		inv.Stdin = strings.NewReader("some magic state")
		err := inv.Run()
		require.NoError(t, err)
	})

	t.Run("NoBuild", func(t *testing.T) {
		t.Parallel()
		client, store := nicloudtest.NewWithDatabase(t, nil)
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, taUser := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		initialState := []byte("initial state")
		r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
			OrganizationID: owner.OrganizationID,
			OwnerID:        taUser.ID,
		}).
			Seed(database.WorkspaceBuild{}).ProvisionerState(initialState).
			Do()
		wantState := []byte("updated state")
		stateFile, err := os.CreateTemp(t.TempDir(), "")
		require.NoError(t, err)
		_, err = stateFile.Write(wantState)
		require.NoError(t, err)
		err = stateFile.Close()
		require.NoError(t, err)

		inv, root := clitest.New(t, "state", "push", "--no-build", r.Workspace.Name, stateFile.Name())
		clitest.SetupConfig(t, templateAdmin, root)
		var stdout bytes.Buffer
		inv.Stdout = &stdout
		err = inv.Run()
		require.NoError(t, err)
		require.Contains(t, stdout.String(), "State updated successfully")

		// Verify the state was updated by pulling it.
		inv, root = clitest.New(t, "state", "pull", r.Workspace.Name)
		var gotState bytes.Buffer
		inv.Stdout = &gotState
		clitest.SetupConfig(t, templateAdmin, root)
		err = inv.Run()
		require.NoError(t, err)
		require.Equal(t, wantState, bytes.TrimSpace(gotState.Bytes()))

		// Verify no new build was created.
		builds, err := store.GetWorkspaceBuildsByWorkspaceID(dbauthz.AsSystemRestricted(context.Background()), database.GetWorkspaceBuildsByWorkspaceIDParams{
			WorkspaceID: r.Workspace.ID,
		})
		require.NoError(t, err)
		require.Len(t, builds, 1, "expected only the initial build, no new build should be created")
	})
}
