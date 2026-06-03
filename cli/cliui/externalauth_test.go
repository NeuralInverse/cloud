package cliui_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/NeuralInverse/cloud/v2/cli/cliui"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/NeuralInverse/cloud/v2/testutil/expecter"
	"github.com/coder/serpent"
)

func TestExternalAuth(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	defer cancel()

	cmd := &serpent.Command{
		Handler: func(inv *serpent.Invocation) error {
			var fetched atomic.Bool
			return cliui.ExternalAuth(inv.Context(), inv.Stdout, cliui.ExternalAuthOptions{
				Fetch: func(ctx context.Context) ([]nicloudsdk.TemplateVersionExternalAuth, error) {
					defer fetched.Store(true)
					return []nicloudsdk.TemplateVersionExternalAuth{{
						ID:              "github",
						DisplayName:     "GitHub",
						Type:            nicloudsdk.EnhancedExternalAuthProviderGitHub.String(),
						Authenticated:   fetched.Load(),
						AuthenticateURL: "https://example.com/gitauth/github",
					}}, nil
				},
				FetchInterval: time.Millisecond,
			})
		},
	}

	inv := cmd.Invoke().WithContext(ctx)
	stdout := expecter.NewAttachedToInvocation(t, inv)

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := inv.Run()
		assert.NoError(t, err)
	}()
	stdout.ExpectMatchContext(ctx, "You must authenticate with")
	stdout.ExpectMatchContext(ctx, "https://example.com/gitauth/github")
	stdout.ExpectMatchContext(ctx, "Successfully authenticated with GitHub")
	<-done
}
