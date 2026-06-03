package chatopenai_test

import (
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd/chatopenai"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestWebSearchToolDisabled(t *testing.T) {
	t.Parallel()

	disabled := false

	tests := []struct {
		name    string
		options *nicloudsdk.ChatModelOpenAIProviderOptions
	}{
		{
			name: "NilOptions",
		},
		{
			name:    "NilWebSearchEnabled",
			options: &nicloudsdk.ChatModelOpenAIProviderOptions{},
		},
		{
			name: "WebSearchDisabled",
			options: &nicloudsdk.ChatModelOpenAIProviderOptions{
				WebSearchEnabled: &disabled,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tool, ok := chatopenai.WebSearchTool(tt.options)
			require.False(t, ok)
			require.Nil(t, tool)
		})
	}
}

func TestWebSearchTool(t *testing.T) {
	t.Parallel()

	enabled := true
	searchContextSize := "high"
	allowedDomains := []string{"example.com", "cloud.neuralinverse.com"}

	tests := []struct {
		name    string
		options *nicloudsdk.ChatModelOpenAIProviderOptions
		want    map[string]any
	}{
		{
			name: "NoExtraFields",
			options: &nicloudsdk.ChatModelOpenAIProviderOptions{
				WebSearchEnabled: &enabled,
			},
			want: map[string]any{},
		},
		{
			name: "SearchContextSize",
			options: &nicloudsdk.ChatModelOpenAIProviderOptions{
				WebSearchEnabled:  &enabled,
				SearchContextSize: &searchContextSize,
			},
			want: map[string]any{
				"search_context_size": searchContextSize,
			},
		},
		{
			name: "AllowedDomains",
			options: &nicloudsdk.ChatModelOpenAIProviderOptions{
				WebSearchEnabled: &enabled,
				AllowedDomains:   allowedDomains,
			},
			want: map[string]any{
				"allowed_domains": allowedDomains,
			},
		},
		{
			name: "BothFields",
			options: &nicloudsdk.ChatModelOpenAIProviderOptions{
				WebSearchEnabled:  &enabled,
				SearchContextSize: &searchContextSize,
				AllowedDomains:    allowedDomains,
			},
			want: map[string]any{
				"search_context_size": searchContextSize,
				"allowed_domains":     allowedDomains,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tool, ok := chatopenai.WebSearchTool(tt.options)
			require.True(t, ok)

			providerTool, ok := tool.(fantasy.ProviderDefinedTool)
			require.True(t, ok)
			require.Equal(t, "web_search", providerTool.ID)
			require.Equal(t, "web_search", providerTool.Name)
			require.NotNil(t, providerTool.Args)
			require.Equal(t, tt.want, providerTool.Args)
		})
	}
}
