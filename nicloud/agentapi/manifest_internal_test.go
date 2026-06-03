package agentapi

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/workspaceapps/appurl"
)

func Test_vscodeProxyURI(t *testing.T) {
	t.Parallel()

	niAccessURL, err := url.Parse("https://cloud.neuralinverse.com")
	require.NoError(t, err)

	accessURLWithPort, err := url.Parse("https://cloud.neuralinverse.com:8080")
	require.NoError(t, err)

	basicApp := appurl.ApplicationURL{
		Prefix:        "prefix",
		AppSlugOrPort: "slug",
		AgentName:     "agent",
		WorkspaceName: "workspace",
		Username:      "user",
	}

	cases := []struct {
		Name        string
		App         appurl.ApplicationURL
		AccessURL   *url.URL
		AppHostname string
		Expected    string
	}{
		{
			Name:        "NoHostname",
			AccessURL:   niAccessURL,
			AppHostname: "",
			App:         basicApp,
			Expected:    "",
		},
		{
			Name:        "NoHostnameAccessURLPort",
			AccessURL:   accessURLWithPort,
			AppHostname: "",
			App:         basicApp,
			Expected:    "",
		},
		{
			Name:        "Hostname",
			AccessURL:   niAccessURL,
			AppHostname: "*.apps.cloud.neuralinverse.com",
			App:         basicApp,
			Expected:    fmt.Sprintf("https://%s.apps.cloud.neuralinverse.com", basicApp.String()),
		},
		{
			Name:        "HostnameWithAccessURLPort",
			AccessURL:   accessURLWithPort,
			AppHostname: "*.apps.cloud.neuralinverse.com",
			App:         basicApp,
			Expected:    fmt.Sprintf("https://%s.apps.cloud.neuralinverse.com:%s", basicApp.String(), accessURLWithPort.Port()),
		},
		{
			Name:        "HostnameWithPort",
			AccessURL:   niAccessURL,
			AppHostname: "*.apps.cloud.neuralinverse.com:4444",
			App:         basicApp,
			Expected:    fmt.Sprintf("https://%s.apps.cloud.neuralinverse.com:%s", basicApp.String(), "4444"),
		},
		{
			// Port from hostname takes precedence over access url port.
			Name:        "HostnameWithPortAccessURLWithPort",
			AccessURL:   accessURLWithPort,
			AppHostname: "*.apps.cloud.neuralinverse.com:4444",
			App:         basicApp,
			Expected:    fmt.Sprintf("https://%s.apps.cloud.neuralinverse.com:%s", basicApp.String(), "4444"),
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			require.NotNilf(t, c.AccessURL, "AccessURL is required")

			output := vscodeProxyURI(c.App, c.AccessURL, c.AppHostname)
			require.Equal(t, c.Expected, output)
		})
	}
}
