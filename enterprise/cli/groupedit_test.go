package cli_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/cli/cliui"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
	"github.com/coder/pretty"
)

func TestGroupEdit(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID, rbac.RoleUserAdmin())

		_, user1 := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID)
		_, user2 := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID)
		_, user3 := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID)

		group := nicloudtest.CreateGroup(t, client, admin.OrganizationID, "alpha", user3)

		expectedName := "beta"

		inv, conf := newCLI(
			t,
			"groups", "edit", group.Name,
			"--name", expectedName,
			"--avatar-url", "https://example.com",
			"-a", user1.ID.String(),
			"-a", user2.Email,
			"-r", user3.ID.String(),
		)

		stdout := expecter.NewAttachedToInvocation(t, inv)
		clitest.SetupConfig(t, anotherClient, conf)
		ctx := testutil.Context(t, testutil.WaitMedium)

		err := inv.Run()
		require.NoError(t, err)

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Successfully patched group %s", pretty.Sprint(cliui.DefaultStyles.Keyword, expectedName)))
	})

	t.Run("InvalidUserInput", func(t *testing.T) {
		t.Parallel()

		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})

		// Create a group with no members.
		group := nicloudtest.CreateGroup(t, client, admin.OrganizationID, "alpha")

		inv, conf := newCLI(
			t,
			"groups", "edit", group.Name,
			"-a", "foo",
		)

		clitest.SetupConfig(t, client, conf) //nolint:gocritic // intentional usage of owner

		err := inv.Run()
		require.ErrorContains(t, err, "must be a valid UUID or email address")
	})

	t.Run("NoArg", func(t *testing.T) {
		t.Parallel()

		client, user := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, user.OrganizationID, rbac.RoleUserAdmin())

		inv, conf := newCLI(t, "groups", "edit")
		clitest.SetupConfig(t, anotherClient, conf)

		err := inv.Run()
		require.ErrorContains(t, err, "wanted 1 args but got 0")
	})
}
