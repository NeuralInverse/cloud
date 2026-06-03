package nicloud_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/coder/serpent"
)

func TestAIBridgeProxyCertificateRetrieval(t *testing.T) {
	t.Parallel()

	t.Run("DisabledReturns404", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.AI.BridgeConfig.Enabled = serpent.Bool(true)
		// Proxy is disabled by default, so we don't need to set it explicitly.
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAIBridge: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitLong)

		// Make a request to the proxy CA cert endpoint.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.URL.String()+"/api/v2/aibridge/proxy/ca-cert.pem", nil)
		require.NoError(t, err)
		req.Header.Set(nicloudsdk.SessionTokenHeader, client.SessionToken())

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("RequiresLicenseFeature", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.AI.BridgeConfig.Enabled = serpent.Bool(true)
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				// No aibridge feature.
				Features: license.Features{},
			},
		})

		ctx := testutil.Context(t, testutil.WaitLong)

		// Make a request to the proxy CA cert endpoint.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.URL.String()+"/api/v2/aibridge/proxy/ca-cert.pem", nil)
		require.NoError(t, err)
		req.Header.Set(nicloudsdk.SessionTokenHeader, client.SessionToken())

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("RequiresAuthentication", func(t *testing.T) {
		t.Parallel()

		dv := nicloudtest.DeploymentValues(t)
		dv.AI.BridgeConfig.Enabled = serpent.Bool(true)
		client, _ := nicloudenttest.New(t, &nicloudenttest.Options{
			Options: &nicloudtest.Options{
				DeploymentValues: dv,
			},
			LicenseOptions: &nicloudenttest.LicenseOptions{
				Features: license.Features{
					nicloudsdk.FeatureAIBridge: 1,
				},
			},
		})

		ctx := testutil.Context(t, testutil.WaitLong)

		// Make a request to the proxy CA cert endpoint without authentication.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.URL.String()+"/api/v2/aibridge/proxy/ca-cert.pem", nil)
		require.NoError(t, err)

		// No session token header set.
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
