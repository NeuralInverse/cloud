package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestEnterpriseListTemplates(t *testing.T) {
	t.Parallel()

	t.Run("MultiOrg", func(t *testing.T) {
		t.Parallel()

		client, owner := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				IncludeProvisionerDaemon: true,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureMultipleOrganizations:      1,
					nicloudsdk.FeatureExternalProvisionerDaemons: 1,
				},
			},
		})

		// Template in the first organization
		firstVersion := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, firstVersion.ID)
		_ = nicloudtest.CreateTemplate(t, client, owner.OrganizationID, firstVersion.ID)

		secondOrg := nicloudenttest.CreateOrganization(t, client, nicloudenttest.CreateOrganizationOptions{
			IncludeProvisionerDaemon: true,
		})
		secondVersion := nicloudtest.CreateTemplateVersion(t, client, secondOrg.ID, nil)
		_ = nicloudtest.CreateTemplate(t, client, secondOrg.ID, secondVersion.ID)

		// Create a site wide template admin
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())

		inv, root := clitest.New(t, "templates", "list", "--output=json")
		clitest.SetupConfig(t, templateAdmin, root)

		ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancelFunc()

		out := bytes.NewBuffer(nil)
		inv.Stdout = out
		err := inv.WithContext(ctx).Run()
		require.NoError(t, err)

		var templates []nicloudsdk.Template
		require.NoError(t, json.Unmarshal(out.Bytes(), &templates))
		require.Len(t, templates, 2)
	})
}
