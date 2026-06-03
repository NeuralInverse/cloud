package cli_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/cli/cliui"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
	"github.com/coder/pretty"
)

func TestTemplateDelete(t *testing.T) {
	t.Parallel()

	t.Run("Ok", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		ctx := testutil.Context(t, testutil.WaitMedium)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)

		inv, root := clitest.New(t, "templates", "delete", template.Name)

		clitest.SetupConfig(t, templateAdmin, root)
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)

		execDone := make(chan error)
		go func() {
			execDone <- inv.Run()
		}()

		stdout.ExpectMatchContext(ctx, fmt.Sprintf("Delete these templates: %s?", pretty.Sprint(cliui.DefaultStyles.Code, template.Name)))
		stdin.WriteLine("yes")

		require.NoError(t, <-execDone)

		_, err := client.Template(context.Background(), template.ID)
		require.Error(t, err, "template should not exist")
	})

	t.Run("Multiple --yes", func(t *testing.T) {
		t.Parallel()

		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		templates := []nicloudsdk.Template{}
		templateNames := []string{}
		for i := 0; i < 3; i++ {
			version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
			_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
			template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
			templates = append(templates, template)
			templateNames = append(templateNames, template.Name)
		}

		inv, root := clitest.New(t, append([]string{"templates", "delete", "--yes"}, templateNames...)...)
		clitest.SetupConfig(t, templateAdmin, root)
		require.NoError(t, inv.Run())

		for _, template := range templates {
			_, err := client.Template(context.Background(), template.ID)
			require.Error(t, err, "template should not exist")
		}
	})

	t.Run("Multiple prompted", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		ctx := testutil.Context(t, testutil.WaitMedium)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		templates := []nicloudsdk.Template{}
		templateNames := []string{}
		for i := 0; i < 3; i++ {
			version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
			_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
			template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)
			templates = append(templates, template)
			templateNames = append(templateNames, template.Name)
		}

		inv, root := clitest.New(t, append([]string{"templates", "delete"}, templateNames...)...)
		clitest.SetupConfig(t, templateAdmin, root)
		stdout := expecter.NewAttachedToInvocation(t, inv)
		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)

		execDone := make(chan error)
		go func() {
			execDone <- inv.Run()
		}()

		stdout.ExpectMatchContext(ctx,
			fmt.Sprintf("Delete these templates: %s?",
				pretty.Sprint(cliui.DefaultStyles.Code, strings.Join(templateNames, ", "))))
		stdin.WriteLine("yes")

		require.NoError(t, <-execDone)

		for _, template := range templates {
			_, err := client.Template(context.Background(), template.ID)
			require.Error(t, err, "template should not exist")
		}
	})

	t.Run("Selector", func(t *testing.T) {
		t.Parallel()

		logger := testutil.Logger(t)
		client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
		owner := nicloudtest.CreateFirstUser(t, client)
		templateAdmin, _ := nicloudtest.CreateAnotherUser(t, client, owner.OrganizationID, rbac.RoleTemplateAdmin())
		version := nicloudtest.CreateTemplateVersion(t, client, owner.OrganizationID, nil)
		_ = nicloudtest.AwaitTemplateVersionJobCompleted(t, client, version.ID)
		template := nicloudtest.CreateTemplate(t, client, owner.OrganizationID, version.ID)

		inv, root := clitest.New(t, "templates", "delete")
		clitest.SetupConfig(t, templateAdmin, root)

		stdin := testutil.NewWriterAttachedToInvocation(t, logger.Named("stdin"), inv)

		execDone := make(chan error)
		go func() {
			execDone <- inv.Run()
		}()

		stdin.WriteLine("yes")
		require.NoError(t, <-execDone)

		_, err := client.Template(context.Background(), template.ID)
		require.Error(t, err, "template should not exist")
	})
}
