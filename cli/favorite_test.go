package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
)

func TestFavoriteUnfavorite(t *testing.T) {
	t.Parallel()

	var (
		client, db           = nicloudtest.NewWithDatabase(t, nil)
		owner                = nicloudtest.CreateFirstUser(t, client)
		memberClient, member = nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		ws                   = dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{OwnerID: member.ID, OrganizationID: owner.OrganizationID}).Do()
	)

	inv, root := clitest.New(t, "favorite", ws.Workspace.Name)
	clitest.SetupConfig(t, memberClient, root)

	var buf bytes.Buffer
	inv.Stdout = &buf
	err := inv.Run()
	require.NoError(t, err)

	updated := nicloudtest.MustWorkspace(t, memberClient, ws.Workspace.ID)
	require.True(t, updated.Favorite)

	buf.Reset()

	inv, root = clitest.New(t, "unfavorite", ws.Workspace.Name)
	clitest.SetupConfig(t, memberClient, root)
	inv.Stdout = &buf
	err = inv.Run()
	require.NoError(t, err)
	updated = nicloudtest.MustWorkspace(t, memberClient, ws.Workspace.ID)
	require.False(t, updated.Favorite)
}
