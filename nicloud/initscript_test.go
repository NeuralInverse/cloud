package nicloud_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestInitScript(t *testing.T) {
	t.Parallel()

	// Single instance shared across all sub-tests. All operations
	// are read-only (fetching init scripts) so parallel execution
	// is safe.
	client := nicloudtest.New(t, nil)

	t.Run("OK Windows amd64", func(t *testing.T) {
		t.Parallel()
		script, err := client.InitScript(context.Background(), "windows", "amd64")
		require.NoError(t, err)
		require.NotEmpty(t, script)
		require.Contains(t, script, "$env:NEURALINVERSE_AGENT_AUTH = \"token\"")
		require.Contains(t, script, "/bin/neuralinverse-windows-amd64.exe")
	})

	t.Run("OK Windows arm64", func(t *testing.T) {
		t.Parallel()
		script, err := client.InitScript(context.Background(), "windows", "arm64")
		require.NoError(t, err)
		require.NotEmpty(t, script)
		require.Contains(t, script, "$env:NEURALINVERSE_AGENT_AUTH = \"token\"")
		require.Contains(t, script, "/bin/neuralinverse-windows-arm64.exe")
	})

	t.Run("OK Linux amd64", func(t *testing.T) {
		t.Parallel()
		script, err := client.InitScript(context.Background(), "linux", "amd64")
		require.NoError(t, err)
		require.NotEmpty(t, script)
		require.Contains(t, script, "export NEURALINVERSE_AGENT_AUTH=\"token\"")
		require.Contains(t, script, "/bin/neuralinverse-linux-amd64")
	})

	t.Run("OK Linux arm64", func(t *testing.T) {
		t.Parallel()
		script, err := client.InitScript(context.Background(), "linux", "arm64")
		require.NoError(t, err)
		require.NotEmpty(t, script)
		require.Contains(t, script, "export NEURALINVERSE_AGENT_AUTH=\"token\"")
		require.Contains(t, script, "/bin/neuralinverse-linux-arm64")
	})

	t.Run("BadRequest", func(t *testing.T) {
		t.Parallel()
		_, err := client.InitScript(context.Background(), "darwin", "armv7")
		require.Error(t, err)
		var apiErr *nicloudsdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
		require.Equal(t, "Unknown os/arch: darwin/armv7", apiErr.Message)
	})
}
