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

func TestCreateGroup(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		client, admin := nicloudenttest.New(t, &nicloudenttest.Options{LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{
				nicloudsdk.FeatureTemplateRBAC: 1,
			},
		}})
		anotherClient, _ := nicloudtest.CreateAnotherUser(t, client, admin.OrganizationID, rbac.RoleUserAdmin())

		var (
			groupName = "test"
			avatarURL = "https://example.com"
		)

		inv, conf := newCLI(t, "groups",
			"create", groupName,
			"--avatar-url", avatarURL,
		)

		stdout := expecter.NewAttachedToInvocation(t, inv)
		clitest.SetupConfig(t, anotherClient, conf)
		ctx := testutil.Context(t, testutil.WaitMedium)

		err := inv.Run()
		require.NoError(t, err)

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Successfully created group %s!", pretty.Sprint(cliui.DefaultStyles.Keyword, groupName)))
	})
}
