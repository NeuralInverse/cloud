package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestTemplateList(t *testing.T) {
	t.Parallel()
	t.Run("ListTemplates", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		firstVersion := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, firstVersion.ID)
		firstTemplate := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, firstVersion.ID)

		secondVersion := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, secondVersion.ID)
		secondTemplate := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, secondVersion.ID)

		inv, root := clitest.New(t, "templates", "list")
		clitest.SetupConfig(t, templateAdmin, root)

		stdout := expecter.NewAttachedToInvocation(t, inv)

		ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancelFunc()

		errC := make(chan error)
		go func() {
			errC <- inv.WithContext(ctx).Run()
		}()

		// expect that templates are listed alphabetically
		templatesList := []string{firstTemplate.Name, secondTemplate.Name}
		slices.Sort(templatesList)

		require.NoError(t, <-errC)

		for _, name := range templatesList {
			stdout.ExpectMatchContext(ctx, name)
		}
	})
	t.Run("ListTemplatesJSON", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		firstVersion := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, firstVersion.ID)
		_ = nicloudtest.CreateTemplate(t, client, owner.OrganizationID, firstVersion.ID)

		secondVersion := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, secondVersion.ID)
		_ = nicloudtest.CreateTemplate(t, client, owner.OrganizationID, secondVersion.ID)

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
	t.Run("NoTemplates", func(t *testing.T) {
		t.Parallel()
		client := nicloudtest.New(t, &nicloudtest.Options{})
		owner := nicloudtest.CreateFirstUser(t, client)

		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())

		inv, root := clitest.New(t, "templates", "list")
		clitest.SetupConfig(t, templateAdmin, root)

		stdout := expecter.NewAttachedToInvocation(t, inv)

		ctx, cancelFunc := context.WithTimeout(context.Background(), testutil.WaitLong)
		defer cancelFunc()

		errC := make(chan error)
		go func() {
			errC <- inv.WithContext(ctx).Run()
		}()

		require.NoError(t, <-errC)

		stdout.ExpectMatchContext(ctx, "No templates found")
		stdout.ExpectMatchContext(ctx, "Create one:")
	})
}
