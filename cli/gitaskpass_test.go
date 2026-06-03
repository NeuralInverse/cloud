package cli_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/cli/clitest"
	"github.com/NeuralInverse/cloud/v2/cli/cliui"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/agentsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
)

func TestGitAskpass(t *testing.T) {
	t.Parallel()
	t.Run("UsernameAndPassword", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpapi.Write(context.Background(), w, http.StatusOK, agentsdk.ExternalAuthResponse{
				Username: "something",
				Password: "bananas",
			})
		}))
		t.Cleanup(srv.Close)
		url := srv.URL
		inv, _ := clitest.New(t, "--agent-url", url, "Username for 'https://github.com':")
		inv.Environ.Set("GIT_PREFIX", "/")
		inv.Environ.Set("NEURALINVERSE_AGENT_TOKEN", "fake-token")
		stdout := expecter.NewAttachedToInvocation(t, inv)
		clitest.Start(t, inv)
		stdout.ExpectMatchContext(ctx, "something")

		inv, _ = clitest.New(t, "--agent-url", url, "Password for 'https://potato@github.com':")
		inv.Environ.Set("GIT_PREFIX", "/")
		inv.Environ.Set("NEURALINVERSE_AGENT_TOKEN", "fake-token")
		stdout = expecter.NewAttachedToInvocation(t, inv)
		clitest.Start(t, inv)
		stdout.ExpectMatchContext(ctx, "bananas")
	})

	t.Run("NoHost", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitMedium)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			httpapi.Write(context.Background(), w, http.StatusNotFound, nicloudsdk.Response{
				Message: "Nope!",
			})
		}))
		t.Cleanup(srv.Close)
		url := srv.URL
		inv, _ := clitest.New(t, "--agent-url", url, "--no-open", "Username for 'https://github.com':")
		inv.Environ.Set("GIT_PREFIX", "/")
		inv.Environ.Set("NEURALINVERSE_AGENT_TOKEN", "fake-token")
		stdout := expecter.NewAttachedToInvocation(t, inv)
		err := inv.Run()
		require.ErrorIs(t, err, cliui.ErrCanceled)
		stdout.ExpectMatchContext(ctx, "Nope!")
	})

	t.Run("Poll", func(t *testing.T) {
		t.Parallel()
		ctx := testutil.Context(t, testutil.WaitShort)
		resp := atomic.Pointer[agentsdk.ExternalAuthResponse]{}
		resp.Store(&agentsdk.ExternalAuthResponse{
			URL: "https://something.org",
		})
		poll := make(chan struct{}, 10)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val := resp.Load()
			if r.URL.Query().Has("listen") {
				poll <- struct{}{}
				if val.URL != "" {
					httpapi.Write(context.Background(), w, http.StatusInternalServerError, val)
					return
				}
			}
			httpapi.Write(context.Background(), w, http.StatusOK, val)
		}))
		t.Cleanup(srv.Close)
		url := srv.URL

		inv, _ := clitest.New(t, "--agent-url", url, "--no-open", "Username for 'https://github.com':")
		inv.Environ.Set("GIT_PREFIX", "/")
		inv.Environ.Set("NEURALINVERSE_AGENT_TOKEN", "fake-token")
		var stdout, stderr *expecter.Expecter
		stdout, inv.Stdout = expecter.NewPiped(t)
		stderr, inv.Stderr = expecter.NewPiped(t)
		go func() {
			err := inv.Run()
			assert.NoError(t, err)
		}()
		testutil.RequireReceive(ctx, t, poll)
		stderr.ExpectMatchContext(ctx, "Open the following URL to authenticate")
		resp.Store(&agentsdk.ExternalAuthResponse{
			Username: "username",
			Password: "password",
		})
		stdout.ExpectMatchContext(ctx, "username")
	})
}
