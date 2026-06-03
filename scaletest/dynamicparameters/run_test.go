package dynamicparameters_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/scaletest/dynamicparameters"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestRun(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitLong)

	client := nicloudtest.New(t, &nicloudtest.Options{IncludeProvisionerDaemon: true})
	client.SetLogger(testutil.Logger(t).Leveled(slog.LevelDebug))
	first := nicloudtest.CreateFirstUser(t, client)
	userClient, _ := nicloudtest.CreateAnotherUser(t, client, first.OrganizationID)
	orgID := first.OrganizationID

	dynamicParametersTerraformSource, err := dynamicparameters.TemplateContent()
	require.NoError(t, err)

	template, version := nicloudtest.DynamicParameterTemplate(t, client, orgID, nicloudtest.DynamicParameterTemplateParams{
		MainTF:         dynamicParametersTerraformSource,
		Plan:           nil,
		ModulesArchive: nil,
		StaticParams:   nil,
		ExtraFiles:     dynamicparameters.GetModuleFiles(),
	})

	reg := prometheus.NewRegistry()
	cfg := dynamicparameters.Config{
		TemplateVersion:   version.ID,
		Metrics:           dynamicparameters.NewMetrics(reg, "template", "test_label_name"),
		MetricLabelValues: []string{template.Name, "test_label_value"},
	}
	runner := dynamicparameters.NewRunner(userClient, cfg)
	var logs strings.Builder
	err = runner.Run(ctx, t.Name(), &logs)
	t.Log("Runner logs:\n\n" + logs.String())
	require.NoError(t, err)
}
