package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog/v3"
	"cdr.dev/slog/v3/sloggers/slogtest"
	"github.com/NeuralInverse/cloud/v2/aibridge"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
	"github.com/coder/serpent"
)

func TestReadAIProvidersFromEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		env         []string
		expected    []nicloudsdk.AIProviderConfig
		errContains string
	}{
		{
			name: "Empty",
			env:  []string{"HOME=/home/frodo"},
		},
		{
			name: "SingleProvider",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=anthropic-zdr",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEY=sk-ant-xxx",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BASE_URL=https://api.anthropic.com/",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{
					Type:    aibridge.ProviderAnthropic,
					Name:    "anthropic-zdr",
					Keys:    []string{"sk-ant-xxx"},
					BaseURL: "https://api.anthropic.com/",
				},
			},
		},
		{
			name: "SingleProviderAIGatewayPrefix",
			env: []string{
				"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_NAME=anthropic-zdr",
				"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_KEY=sk-ant-xxx",
				"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_BASE_URL=https://api.anthropic.com/",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{
					Type:    aibridge.ProviderAnthropic,
					Name:    "anthropic-zdr",
					Keys:    []string{"sk-ant-xxx"},
					BaseURL: "https://api.anthropic.com/",
				},
			},
		},
		{
			name: "MultipleProvidersSameType",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=anthropic-us",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_NAME=anthropic-eu",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_BASE_URL=https://eu.api.anthropic.com/",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{Type: aibridge.ProviderAnthropic, Name: "anthropic-us"},
				{Type: aibridge.ProviderAnthropic, Name: "anthropic-eu", BaseURL: "https://eu.api.anthropic.com/"},
			},
		},
		{
			name: "DefaultName",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{Type: aibridge.ProviderOpenAI, Name: aibridge.ProviderOpenAI},
			},
		},
		{
			name: "MixedTypes",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=anthropic-main",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_2_TYPE=copilot",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_2_NAME=copilot-custom",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_2_BASE_URL=https://custom.copilot.com",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{Type: aibridge.ProviderAnthropic, Name: "anthropic-main"},
				{Type: aibridge.ProviderOpenAI, Name: aibridge.ProviderOpenAI},
				{Type: aibridge.ProviderCopilot, Name: "copilot-custom", BaseURL: "https://custom.copilot.com"},
			},
		},
		{
			name: "BedrockFields",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=anthropic-bedrock",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_REGION=us-west-2",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY=AKID",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRET=secret",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_MODEL=anthropic.claude-3-sonnet",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_SMALL_FAST_MODEL=anthropic.claude-3-haiku",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_BASE_URL=https://bedrock.us-west-2.amazonaws.com",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{
					Type:                    aibridge.ProviderAnthropic,
					Name:                    "anthropic-bedrock",
					BedrockRegion:           "us-west-2",
					BedrockAccessKeys:       []string{"AKID"},
					BedrockAccessKeySecrets: []string{"secret"},
					BedrockModel:            "anthropic.claude-3-sonnet",
					BedrockSmallFastModel:   "anthropic.claude-3-haiku",
					BedrockBaseURL:          "https://bedrock.us-west-2.amazonaws.com",
				},
			},
		},
		{
			name: "OutOfOrderIndices",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_NAME=second",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=first",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{Type: aibridge.ProviderOpenAI, Name: "first"},
				{Type: aibridge.ProviderAnthropic, Name: "second"},
			},
		},
		{
			name:        "SkippedIndex",
			env:         []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai", "NEURALINVERSE_AIBRIDGE_PROVIDER_2_TYPE=anthropic"},
			errContains: "skipped",
		},
		{
			name:        "InvalidKey",
			env:         []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_XXX_TYPE=openai"},
			errContains: "parse number",
		},
		{
			name:        "MissingType",
			env:         []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=my-provider", "NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEY=sk-xxx"},
			errContains: "TYPE is required",
		},
		{
			name:        "InvalidType",
			env:         []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=gemini"},
			errContains: "unknown TYPE",
		},
		{
			name: "DuplicateExplicitNames",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=my-provider",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_1_NAME=my-provider",
			},
			errContains: "duplicate NAME",
		},
		{
			name:        "DuplicateDefaultNames",
			env:         []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic", "NEURALINVERSE_AIBRIDGE_PROVIDER_1_TYPE=anthropic"},
			errContains: "duplicate NAME",
		},
		{
			name:        "BedrockFieldsOnNonAnthropic",
			env:         []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai", "NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_REGION=us-west-2"},
			errContains: "BEDROCK_* fields are only supported with TYPE",
		},
		{
			name: "IgnoresUnrelatedEnvVars",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_OPENAI_KEY=should-be-ignored",
				"NEURALINVERSE_AIBRIDGE_ANTHROPIC_KEY=also-ignored",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEY=sk-xxx",
				"SOME_OTHER_VAR=hello",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{Type: aibridge.ProviderOpenAI, Name: aibridge.ProviderOpenAI, Keys: []string{"sk-xxx"}},
			},
		},
		{
			// KEYS is a plural alias for KEY.
			name: "PluralKeysAlias",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-ant-xxx",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{
					Type: aibridge.ProviderAnthropic,
					Name: aibridge.ProviderAnthropic,
					Keys: []string{"sk-ant-xxx"},
				},
			},
		},
		{
			// BEDROCK_ACCESS_KEYS and BEDROCK_ACCESS_KEY_SECRETS are
			// plural aliases for their singular counterparts.
			name: "PluralBedrockAliases",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEYS=AKID",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRETS=secret",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{
					Type:                    aibridge.ProviderAnthropic,
					Name:                    aibridge.ProviderAnthropic,
					BedrockAccessKeys:       []string{"AKID"},
					BedrockAccessKeySecrets: []string{"secret"},
				},
			},
		},
		{
			// An Anthropic provider can't use both a bearer token
			// (KEYS) and Bedrock (BEDROCK_*); they're mutually
			// exclusive authentication modes.
			name: "AnthropicKeysAndBedrockConflict",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-ant-xxx",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_REGION=us-east-1",
			},
			errContains: "KEY/KEYS and BEDROCK_* fields are mutually exclusive",
		},
		{
			name: "ConflictKeyAndKeys",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEY=sk-single",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-multi",
			},
			errContains: "KEY and KEYS are mutually exclusive",
		},
		{
			name: "ConflictBedrockAccessKeyAndKeys",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY=AKID1",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEYS=AKID2",
			},
			errContains: "BEDROCK_ACCESS_KEY and BEDROCK_ACCESS_KEYS are mutually exclusive",
		},
		{
			name: "ConflictBedrockSecretAndSecrets",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRET=s1",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRETS=s2",
			},
			errContains: "BEDROCK_ACCESS_KEY_SECRET and BEDROCK_ACCESS_KEY_SECRETS are mutually exclusive",
		},
		{
			name: "CopilotRejectsKey",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=copilot",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEY=sk-xxx",
			},
			errContains: "KEY/KEYS are not supported for TYPE",
		},
		{
			name: "CopilotRejectsKeys",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=copilot",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-a,sk-b",
			},
			errContains: "KEY/KEYS are not supported for TYPE",
		},
		{
			name: "MultipleKeysCommaSeparated",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-a,sk-b,sk-c",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{Type: aibridge.ProviderOpenAI, Name: aibridge.ProviderOpenAI, Keys: []string{"sk-a", "sk-b", "sk-c"}},
			},
		},
		{
			name: "KeysWhitespaceTrimmed",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS= sk-a , sk-b ",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{Type: aibridge.ProviderOpenAI, Name: aibridge.ProviderOpenAI, Keys: []string{"sk-a", "sk-b"}},
			},
		},
		{
			name: "KeysEmptyAfterTrim",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-a,,sk-b",
			},
			errContains: "key at index 1 is empty",
		},
		{
			name: "KeysDuplicate",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-a,sk-b,sk-a",
			},
			errContains: "duplicate key at index 2",
		},
		{
			name: "KeysTooMany",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEYS=sk-1,sk-2,sk-3,sk-4,sk-5,sk-6",
			},
			errContains: "too many keys (6), maximum is 5",
		},
		{
			name: "BedrockMultipleKeys",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_REGION=us-west-2",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEYS=AKID1,AKID2",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRETS=secret1,secret2",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{
					Type:                    aibridge.ProviderAnthropic,
					Name:                    aibridge.ProviderAnthropic,
					BedrockRegion:           "us-west-2",
					BedrockAccessKeys:       []string{"AKID1", "AKID2"},
					BedrockAccessKeySecrets: []string{"secret1", "secret2"},
				},
			},
		},
		{
			name: "BedrockKeyCountMismatch",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEYS=AKID1,AKID2",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRET=secret1",
			},
			errContains: "BEDROCK_ACCESS_KEYS count (2) must match BEDROCK_ACCESS_KEY_SECRETS count (1)",
		},
		{
			name: "MixedPrefixesAreNotAllowed",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=anthropic-1",
				"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_NAME=anthropic-2",
			},
			errContains: "cannot mix NEURALINVERSE_AIBRIDGE_PROVIDER_* and NEURALINVERSE_AI_GATEWAY_PROVIDER_* environment variables",
		},
		{
			name: "BedrockTypeHappyPath",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=bedrock",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=bedrock-prod",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_REGION=us-east-1",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY=AKID",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRET=secret",
			},
			expected: []nicloudsdk.AIProviderConfig{
				{
					Type:                    string(database.AiProviderTypeBedrock),
					Name:                    "bedrock-prod",
					BedrockRegion:           "us-east-1",
					BedrockAccessKeys:       []string{"AKID"},
					BedrockAccessKeySecrets: []string{"secret"},
				},
			},
		},
		{
			name:        "BedrockTypeWithoutBedrockFields",
			env:         []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=bedrock", "NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=bedrock-prod"},
			errContains: "requires BEDROCK_* fields to be configured",
		},
		{
			name: "BedrockTypeRejectsAPIKeys",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=bedrock",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_NAME=bedrock-prod",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_REGION=us-east-1",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_KEY=sk-should-fail",
			},
			errContains: "KEY/KEYS are not supported for TYPE",
		},
		{
			name: "BedrockKeysTooMany",
			env: []string{
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=anthropic",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEYS=AKID1,AKID2,AKID3,AKID4,AKID5,AKID6",
				"NEURALINVERSE_AIBRIDGE_PROVIDER_0_BEDROCK_ACCESS_KEY_SECRETS=s1,s2,s3,s4,s5,s6",
			},
			errContains: "too many keys (6), maximum is 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			providers, err := ReadAIProvidersFromEnv(slogtest.Make(t, nil), tt.env)
			if tt.errContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, providers)
		})
	}

	// Cases below need special setup that doesn't fit the table above.

	t.Run("MultiDigitIndices", func(t *testing.T) {
		t.Parallel()
		// Indices 0, 1, 2, ..., 10, verifies that 10 sorts after 2,
		// not between 1 and 2 as a lexicographic sort would do.
		var env []string
		var expected []nicloudsdk.AIProviderConfig
		for i := range 11 {
			env = append(env,
				fmt.Sprintf("NEURALINVERSE_AIBRIDGE_PROVIDER_%d_TYPE=openai", i),
				fmt.Sprintf("NEURALINVERSE_AIBRIDGE_PROVIDER_%d_KEY=sk-%d", i, i),
				fmt.Sprintf("NEURALINVERSE_AIBRIDGE_PROVIDER_%d_NAME=p%d", i, i),
			)
			expected = append(expected, nicloudsdk.AIProviderConfig{
				Type: aibridge.ProviderOpenAI,
				Name: fmt.Sprintf("p%d", i),
				Keys: []string{fmt.Sprintf("sk-%d", i)},
			})
		}
		providers, err := ReadAIProvidersFromEnv(slogtest.Make(t, nil), env)
		require.NoError(t, err)
		require.Equal(t, expected, providers)
	})

	t.Run("UnknownFieldWarnsButSucceeds", func(t *testing.T) {
		t.Parallel()
		// A typo like TYYYPPOO instead of TYPE should not prevent startup;
		// the function logs a warning and continues.
		tests := []struct {
			name             string
			env              []string
			expected         []nicloudsdk.AIProviderConfig
			expectedWarnings []string
		}{
			{
				name: "AIGatewayPrefix",
				env: []string{
					"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_TYPE=openai",
					"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_Name=test",
					"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_TYYYPPOO=openai",
				},
				expected: []nicloudsdk.AIProviderConfig{
					{Type: "openai", Name: "test"},
				},
				expectedWarnings: []string{"NEURALINVERSE_AI_GATEWAY_PROVIDER_0_TYYYPPOO"},
			},
			{
				name: "AIBridgePrefix",
				env: []string{
					"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYPE=openai",
					"NEURALINVERSE_AIBRIDGE_PROVIDER_0_Name=test",
					"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYYYPPOO=openai",
				},
				expected: []nicloudsdk.AIProviderConfig{
					{Type: "openai", Name: "test"},
				},
				expectedWarnings: []string{"NEURALINVERSE_AIBRIDGE_PROVIDER_0_TYYYPPOO"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				sink := testutil.NewFakeSink(t)
				providers, err := ReadAIProvidersFromEnv(sink.Logger(), tt.env)
				require.NoError(t, err)
				require.Equal(t, tt.expected, providers)

				warnings := sink.Entries(func(e slog.SinkEntry) bool {
					return e.Message == "ignoring unknown AI provider field (check for typos)"
				})
				require.Len(t, warnings, len(tt.expectedWarnings))
				for i, want := range tt.expectedWarnings {
					require.Len(t, warnings[i].Fields, 1)
					assert.Equal(t, want, warnings[i].Fields[0].Value)
				}
			})
		}
	})
}

func TestValidateLegacyAIBridgeConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         nicloudsdk.AIBridgeConfig
		errContains string
	}{
		{
			name: "BareAnthropicKey",
			cfg: nicloudsdk.AIBridgeConfig{
				LegacyAnthropic: nicloudsdk.AIBridgeAnthropicConfig{Key: "sk-ant"},
			},
		},
		{
			name: "BareBedrockRegion",
			cfg: nicloudsdk.AIBridgeConfig{
				LegacyBedrock: nicloudsdk.AIBridgeBedrockConfig{Region: "us-east-1"},
			},
		},
		{
			name: "BedrockCredentialsOnly",
			cfg: nicloudsdk.AIBridgeConfig{
				LegacyBedrock: nicloudsdk.AIBridgeBedrockConfig{
					AccessKey:       "AKIA",
					AccessKeySecret: "secret",
				},
			},
		},
		{
			name: "AnthropicKeyAndBedrockConflict",
			cfg: nicloudsdk.AIBridgeConfig{
				LegacyAnthropic: nicloudsdk.AIBridgeAnthropicConfig{Key: "sk-ant"},
				LegacyBedrock: nicloudsdk.AIBridgeBedrockConfig{
					Region:          "us-east-1",
					AccessKey:       "AKIA",
					AccessKeySecret: "secret",
				},
			},
			errContains: "NEURALINVERSE_AIBRIDGE_ANTHROPIC_KEY and NEURALINVERSE_AIBRIDGE_BEDROCK_* are mutually exclusive",
		},
		{
			name: "AnthropicKeyWithBedrockModelDefaultsIsFine",
			cfg: nicloudsdk.AIBridgeConfig{
				LegacyAnthropic: nicloudsdk.AIBridgeAnthropicConfig{Key: "sk-ant"},
				// Model defaults shouldn't trip the conflict; they're
				// always populated in a real deployment.
				LegacyBedrock: nicloudsdk.AIBridgeBedrockConfig{
					Model:          "anthropic.claude-3-5-sonnet",
					SmallFastModel: "anthropic.claude-3-5-haiku",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateLegacyAIBridgeConfig(tt.cfg)
			if tt.errContains == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestWarnIfAIProvidersConfiguredFromEnv(t *testing.T) {
	t.Parallel()

	t.Run("NoProviders", func(t *testing.T) {
		t.Parallel()

		sink := testutil.NewFakeSink(t)
		warnIfAIProvidersConfiguredFromEnv(context.Background(), sink.Logger(), aiGatewayProviderEnvPrefix, nil)

		require.Empty(t, sink.Entries())
	})

	t.Run("EmptyPrefix", func(t *testing.T) {
		t.Parallel()

		sink := testutil.NewFakeSink(t)
		warnIfAIProvidersConfiguredFromEnv(context.Background(), sink.Logger(), "", []nicloudsdk.AIProviderConfig{{Type: "openai", Name: "openai"}})

		require.Empty(t, sink.Entries())
	})

	t.Run("AIGatewayPrefix", func(t *testing.T) {
		t.Parallel()

		sink := testutil.NewFakeSink(t)
		warnIfAIProvidersConfiguredFromEnv(context.Background(), sink.Logger(), aiGatewayProviderEnvPrefix, []nicloudsdk.AIProviderConfig{{Type: "openai", Name: "openai"}})

		entries := sink.Entries(func(e slog.SinkEntry) bool {
			return e.Message == "ai provider environment variables are deprecated for provider management and only seed provider configuration at startup"
		})
		require.Len(t, entries, 1)
		require.Len(t, entries[0].Fields, 2)
		assertFieldValue(t, entries[0].Fields, "env_prefix", aiGatewayProviderEnvPrefix)
		assertFieldValue(t, entries[0].Fields, "replacement", "Manage AI Providers from the Coder UI or HTTP API.")
	})

	t.Run("AIBridgePrefix", func(t *testing.T) {
		t.Parallel()

		sink := testutil.NewFakeSink(t)
		warnIfAIProvidersConfiguredFromEnv(context.Background(), sink.Logger(), aiBridgeProviderEnvPrefix, []nicloudsdk.AIProviderConfig{{Type: "openai", Name: "openai"}})

		entries := sink.Entries(func(e slog.SinkEntry) bool {
			return e.Message == "ai provider environment variables are deprecated for provider management and only seed provider configuration at startup"
		})
		require.Len(t, entries, 1)
		require.Len(t, entries[0].Fields, 2)
		assertFieldValue(t, entries[0].Fields, "env_prefix", aiBridgeProviderEnvPrefix)
		assertFieldValue(t, entries[0].Fields, "replacement", "Manage AI Providers from the Coder UI or HTTP API.")
	})
}

func TestBuildAIProviderFromRowSetsAPIDumpDir(t *testing.T) {
	t.Parallel()

	const dumpDir = "/tmp/coder-aibridge-dumps"

	tests := []struct {
		name         string
		row          database.AIProvider
		expectedType string
	}{
		{
			name: "OpenAI",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeOpenai,
				Name:    "openai",
				BaseUrl: "https://api.openai.com/",
			},
			expectedType: aibridge.ProviderOpenAI,
		},
		{
			name: "Anthropic",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeAnthropic,
				Name:    "anthropic",
				BaseUrl: "https://api.anthropic.com/",
			},
			expectedType: aibridge.ProviderAnthropic,
		},
		{
			name: "Copilot",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeCopilot,
				Name:    "copilot",
				BaseUrl: "https://api.githubcopilot.com/",
			},
			expectedType: aibridge.ProviderCopilot,
		},
		{
			name: "Azure",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeAzure,
				Name:    "azure",
				BaseUrl: "https://example.openai.azure.com/",
			},
			expectedType: aibridge.ProviderOpenAI,
		},
		{
			name: "Google",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeGoogle,
				Name:    "google",
				BaseUrl: "https://generativelanguage.googleapis.com/v1beta/openai/",
			},
			expectedType: aibridge.ProviderOpenAI,
		},
		{
			name: "OpenAICompat",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeOpenaiCompat,
				Name:    "openai-compat",
				BaseUrl: "https://compat.example.com/v1/",
			},
			expectedType: aibridge.ProviderOpenAI,
		},
		{
			name: "OpenRouter",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeOpenrouter,
				Name:    "openrouter",
				BaseUrl: "https://openrouter.ai/api/v1/",
			},
			expectedType: aibridge.ProviderOpenAI,
		},
		{
			name: "Vercel",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeVercel,
				Name:    "vercel",
				BaseUrl: "https://api.v0.dev/v1/",
			},
			expectedType: aibridge.ProviderOpenAI,
		},
		{
			name: "Bedrock",
			row: database.AIProvider{
				Enabled: true,
				Type:    database.AiProviderTypeBedrock,
				Name:    "bedrock",
				BaseUrl: "https://bedrock-runtime.us-east-1.amazonaws.com/",
				Settings: mustMarshalSettings(nicloudsdk.AIProviderSettings{
					Bedrock: &nicloudsdk.AIProviderBedrockSettings{
						Region:          "us-east-1",
						AccessKey:       ptr.Ref("AKID"),
						AccessKeySecret: ptr.Ref("secret"),
					},
				}),
			},
			expectedType: aibridge.ProviderAnthropic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, err := buildAIProviderFromRow(tt.row, nil, nicloudsdk.AIBridgeConfig{
				AllowBYOK:  serpent.Bool(true),
				APIDumpDir: serpent.String(dumpDir),
			})
			require.NoError(t, err)
			assert.Equal(t, dumpDir, provider.APIDumpDir())
			assert.Equal(t, tt.expectedType, provider.Type())
		})
	}
}

func TestBuildAIProviderFromRowBedrockWithoutSettings(t *testing.T) {
	t.Parallel()

	_, err := buildAIProviderFromRow(database.AIProvider{
		Enabled: true,
		Type:    database.AiProviderTypeBedrock,
		Name:    "bedrock-no-settings",
		BaseUrl: "https://bedrock-runtime.us-east-1.amazonaws.com/",
	}, nil, nicloudsdk.AIBridgeConfig{
		AllowBYOK: serpent.Bool(true),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bedrock provider has no bedrock credentials configured")
}

func mustMarshalSettings(s nicloudsdk.AIProviderSettings) sql.NullString {
	data, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return sql.NullString{String: string(data), Valid: true}
}

func assertFieldValue(t *testing.T, fields slog.Map, name string, expected interface{}) {
	t.Helper()
	for _, f := range fields {
		if f.Name == name {
			assert.Equal(t, expected, f.Value)
			return
		}
	}
	t.Errorf("field %q not found", name)
}
