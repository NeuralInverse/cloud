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

func TestGroupDelete(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID, rbac.RoleUserAdmin())

		group := nicloudtest.CreateGroup(t, client, admin.OrganizationID, "alpha")

		inv, conf := newCLI(t,
			"groups", "delete", group.Name,
		)

		stdout := expecter.NewAttachedToInvocation(t, inv)
		ctx := testutil.Context(t, testutil.WaitMedium)
		clitest.SetupConfig(t, anotherClient, conf)

		err := inv.Run()
		require.NoError(t, err)

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Successfully deleted group %s", pretty.Sprint(cliui.DefaultStyles.Keyword, group.Name)))
	})

	t.Run("NoArg", func(t *testing.T) {
		t.Parallel()

		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID, rbac.RoleUserAdmin())

		inv, conf := newCLI(
			t,
			"groups", "delete",
		)

		clitest.SetupConfig(t, anotherClient, conf)

		err := inv.Run()
		require.Error(t, err)
	})
}
