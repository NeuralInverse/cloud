//nolint:testpackage // These tests cover unexported request-patch guards.
package chatprovider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/codersdk"
)

func TestPatchAnthropicThinkingDisplayBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		setting *anthropicThinkingDisplaySetting
		want    string
	}{
		{
			name: "ExplicitDisplayIsInjected",
			body: `{
				"model": "claude-sonnet-4-20250514",
				"thinking": {"type": "adaptive"},
				"max_tokens": 100
			}`,
			setting: &anthropicThinkingDisplaySetting{Display: "omitted"},
			want:    "omitted",
		},
		{
			name: "Opus48DefaultsToSummarized",
			body: `{
				"model": "claude-opus-4-8",
				"thinking": {"type": "adaptive"},
				"max_tokens": 100
			}`,
			want: "summarized",
		},
		{
			name: "ExistingDisplayIsPreserved",
			body: `{
				"model": "claude-opus-4-8",
				"thinking": {"type": "adaptive", "display": "omitted"},
				"max_tokens": 100
			}`,
			want: "omitted",
		},
		{
			name: "ExplicitDisplayOverridesExistingBody",
			body: `{
				"model": "claude-opus-4-8",
				"thinking": {"type": "adaptive", "display": "omitted"},
				"max_tokens": 100
			}`,
			setting: &anthropicThinkingDisplaySetting{Display: "summarized"},
			want:    "summarized",
		},
		{
			name: "NoThinkingIsUnchanged",
			body: `{
				"model": "claude-opus-4-8",
				"max_tokens": 100
			}`,
			want: "",
		},
		{
			name: "OtherModelsDoNotDefault",
			body: `{
				"model": "claude-sonnet-4-20250514",
				"thinking": {"type": "adaptive"},
				"max_tokens": 100
			}`,
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			patched := patchAnthropicThinkingDisplayBody([]byte(tt.body), tt.setting)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(patched, &payload))

			thinking, _ := payload["thinking"].(map[string]any)
			if tt.want == "" {
				if thinking != nil {
					require.NotContains(t, thinking, "display")
				}
				return
			}
			require.Equal(t, tt.want, thinking["display"])
		})
	}
}

func TestContextWithAnthropicThinkingDisplay(t *testing.T) {
	t.Parallel()

	display := " Summarized "
	ctx := ContextWithAnthropicThinkingDisplay(context.Background(), &codersdk.ChatModelProviderOptions{
		Anthropic: &codersdk.ChatModelAnthropicProviderOptions{
			Thinking: &codersdk.ChatModelAnthropicThinkingOptions{Display: &display},
		},
	})

	setting := anthropicThinkingDisplayFromContext(ctx)
	require.NotNil(t, setting)
	require.Equal(t, "summarized", setting.Display)
}
