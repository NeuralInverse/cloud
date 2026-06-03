package cli_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/cli/config"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/enterprise/cli"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/coder/serpent"
)

func newCLI(t *testing.T, args ...string) (*serpent.Invocation, config.Root) {
	var root cli.RootCmd
	cmd, err := root.Command(root.EnterpriseSubcommands())
	require.NoError(t, err)
	return clitest.NewWithCommand(t, cmd, args...)
}

func TestEnterpriseHandlersOK(t *testing.T) {
	t.Parallel()

	var root cli.RootCmd
	cmd, err := root.Command(root.EnterpriseSubcommands())
	require.NoError(t, err)

	clitest.HandlersOK(t, cmd)
}

func TestCheckWarnings(t *testing.T) {
	t.Parallel()

	t.Run("LicenseWarningForPrivilegedRoles", func(t *testing.T) {
		t.Parallel()
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				ExpiresAt: time.Now().Add(time.Hour * 24),
			},
		})

		inv, conf := newCLI(t, "list")

		var buf bytes.Buffer
		inv.Stderr = &buf
		clitest.SetupConfig(t, client, conf) //nolint:gocritic // owners should see this

		err := inv.Run()
		require.NoError(t, err)

		require.Contains(t, buf.String(), "Your license expires in 1 day.")
	})

	t.Run("NoLicenseWarningForRegularUser", func(t *testing.T) {
		t.Parallel()
		adminClient, admin := nicloudenttest.New(t, &nicloudenttest.Options{
			LicenseOptions: &nicloudenttest.LicenseOptions{
				ExpiresAt: time.Now().Add(time.Hour * 24),
			},
		})

		client, _ := nicloudtest.CreateAnotherUser(t, adminClient, admin.OrganizationID)

		inv, conf := newCLI(t, "list")

		var buf bytes.Buffer
		inv.Stderr = &buf
		clitest.SetupConfig(t, client, conf)

		err := inv.Run()
		require.NoError(t, err)

		require.NotContains(t, buf.String(), "Your license expires")
	})
}
