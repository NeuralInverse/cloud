package cli_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestAutoUpdate(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		member, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID)
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
		workspace := nicloudtest.CreateWorkspace(t, member, template.ID)
		nicloudtest.AwaitWorkspaceBuildJobCompleted(t, client, workspace.LatestBuild.ID)
		require.Equal(t, nicloudsdk.AutomaticUpdatesNever, workspace.AutomaticUpdates)

		expectedPolicy := nicloudsdk.AutomaticUpdatesAlways
		inv, root := clitest.New(t, "autoupdate", workspace.Name, string(expectedPolicy))
		clitest.SetupConfig(t, member, root)
		var buf bytes.Buffer
		inv.Stdout = &buf
		err := inv.Run()
		require.NoError(t, err)
		require.Contains(t, buf.String(), fmt.Sprintf("Updated workspace %q auto-update policy to %q", workspace.Name, expectedPolicy))

		workspace = nicloudtest.MustWorkspace(t, client, workspace.ID)
		require.Equal(t, expectedPolicy, workspace.AutomaticUpdates)
	})

	t.Run("InvalidArgs", func(t *testing.T) {
		type testcase struct {
			Name          string
			Args          []string
			ErrorContains string
		}

		cases := []testcase{
			{
				Name:          "NoPolicy",
				Args:          []string{"autoupdate", "ws"},
				ErrorContains: "wanted 2 args but got 1",
			},
			{
				Name:          "InvalidPolicy",
				Args:          []string{"autoupdate", "ws", "sometimes"},
				ErrorContains: `invalid option "sometimes" must be either of`,
			},
		}

		for _, c := range cases {
			t.Run(c.Name, func(t *testing.T) {
				t.Parallel()
				client := nicloudtest.New(t, nil)
				_ = nicloudtest.CreateFirstUser(t, client)

				inv, root := clitest.New(t, c.Args...)
				clitest.SetupConfig(t, client, root)
				err := inv.Run()
				require.Error(t, err)
				require.Contains(t, err.Error(), c.ErrorContains)
			})
		}
	})
}
