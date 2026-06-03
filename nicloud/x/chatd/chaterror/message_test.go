package chaterror_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd/chaterror"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// TestTerminalMessage covers the per-provider "temporarily
// unavailable" copy, the startup-timeout copy, and the generic
// fallback string for its intended (unclassified, non-retryable)
// path.
func TestTerminalMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		kind       nicloudsdk.ChatErrorKind
		provider   string
		retryable  bool
		statusCode int
		want       string
	}{
		{
			name:      "Timeout_Retryable_Anthropic",
			kind:      nicloudsdk.ChatErrorKindTimeout,
			provider:  "anthropic",
			retryable: true,
			want:      "Anthropic is temporarily unavailable.",
		},
		{
			name:      "Timeout_Retryable_OpenAI",
			kind:      nicloudsdk.ChatErrorKindTimeout,
			provider:  "openai",
			retryable: true,
			want:      "OpenAI is temporarily unavailable.",
		},
		{
			name:      "Timeout_Retryable_UnknownProvider",
			kind:      nicloudsdk.ChatErrorKindTimeout,
			provider:  "",
			retryable: true,
			want:      "The AI provider is temporarily unavailable.",
		},
		{
			name:      "Timeout_NotRetryable_NoStatus",
			kind:      nicloudsdk.ChatErrorKindTimeout,
			provider:  "",
			retryable: false,
			want:      "The request timed out before it completed.",
		},
		{
			name:      "StartupTimeout_Anthropic",
			kind:      nicloudsdk.ChatErrorKindStartupTimeout,
			provider:  "anthropic",
			retryable: true,
			want:      "Anthropic did not start responding in time.",
		},
		{
			name:      "StartupTimeout_OpenAI",
			kind:      nicloudsdk.ChatErrorKindStartupTimeout,
			provider:  "openai",
			retryable: true,
			want:      "OpenAI did not start responding in time.",
		},
		{
			// Generic fallback reserved for genuinely
			// unclassified non-retryable failures.
			name:      "Generic_NotRetryable_NoStatus",
			kind:      nicloudsdk.ChatErrorKindGeneric,
			provider:  "",
			retryable: false,
			want:      "The chat request failed unexpectedly.",
		},
		{
			name:      "UsageLimit_OpenAI",
			kind:      nicloudsdk.ChatErrorKindUsageLimit,
			provider:  "openai",
			retryable: false,
			want:      "The usage quota for OpenAI has been exceeded. Check the billing and quota settings for the provider account.",
		},
		{
			name:      "UsageLimit_UnknownProvider",
			kind:      nicloudsdk.ChatErrorKindUsageLimit,
			provider:  "",
			retryable: false,
			want:      "The usage quota for the AI provider has been exceeded. Check the billing and quota settings for the provider account.",
		},
		{
			name:      "MissingKey",
			kind:      nicloudsdk.ChatErrorKindMissingKey,
			provider:  "",
			retryable: false,
			want:      "This conversation was started with an API key that is no longer available. Send your message again to continue.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			classified := chaterror.ClassifiedError{
				Kind:       tt.kind,
				Provider:   tt.provider,
				Retryable:  tt.retryable,
				StatusCode: tt.statusCode,
			}
			// terminalMessage is unexported; round-trip through
			// WithClassification + Classify to exercise it.
			wrapped := chaterror.WithClassification(
				xerrors.New(tt.name),
				classified,
			)
			require.Equal(t, tt.want, chaterror.Classify(wrapped).Message)
		})
	}
}
