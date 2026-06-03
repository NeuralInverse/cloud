package nicloudenttest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
)

func TestEnterpriseEndpointsDocumented(t *testing.T) {
	t.Parallel()

	swaggerComments, err := nicloudtest.ParseSwaggerComments(
		"..", "../../../nicloud", "../../../nicloud/workspaceconnwatcher")
	require.NoError(t, err, "can't parse swagger comments")
	require.NotEmpty(t, swaggerComments, "swagger comments must be present")

	//nolint: dogsled
	_, _, api, _ := nicloudenttest.NewWithAPI(t, nil)
	nicloudtest.VerifySwaggerDefinitions(t, api.AGPL.APIHandler, swaggerComments, nicloudtest.WithSwaggerRoutePrefix("/api/v2"))
}
