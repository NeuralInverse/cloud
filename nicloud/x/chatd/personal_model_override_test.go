package chatd_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestChatPersonalModelOverrideKey(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"chat_personal_model_override:root",
		chatd.ChatPersonalModelOverrideKey(nicloudsdk.ChatPersonalModelOverrideContextRoot),
	)
}

func TestParseChatPersonalModelOverride(t *testing.T) {
	t.Parallel()

	modelConfigID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tests := []struct {
		name        string
		raw         string
		defaultMode nicloudsdk.ChatPersonalModelOverrideMode
		want        chatd.ParsedChatPersonalModelOverride
	}{
		{
			name:        "EmptyUsesDefault",
			raw:         "",
			defaultMode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
			want: chatd.ParsedChatPersonalModelOverride{
				Mode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
			},
		},
		{
			name:        "ChatDefault",
			raw:         string(nicloudsdk.ChatPersonalModelOverrideModeChatDefault),
			defaultMode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
			want: chatd.ParsedChatPersonalModelOverride{
				Mode: nicloudsdk.ChatPersonalModelOverrideModeChatDefault,
			},
		},
		{
			name:        "DeploymentDefault",
			raw:         string(nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault),
			defaultMode: nicloudsdk.ChatPersonalModelOverrideModeChatDefault,
			want: chatd.ParsedChatPersonalModelOverride{
				Mode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
			},
		},
		{
			name:        "Model",
			raw:         "model:" + modelConfigID.String(),
			defaultMode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
			want: chatd.ParsedChatPersonalModelOverride{
				Mode:          nicloudsdk.ChatPersonalModelOverrideModeModel,
				ModelConfigID: modelConfigID,
			},
		},
		{
			name:        "InvalidModelUUID",
			raw:         "model:not-a-uuid",
			defaultMode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
			want: chatd.ParsedChatPersonalModelOverride{
				Mode:      nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
				Malformed: true,
			},
		},
		{
			name:        "UnknownValue",
			raw:         "unknown",
			defaultMode: nicloudsdk.ChatPersonalModelOverrideModeChatDefault,
			want: chatd.ParsedChatPersonalModelOverride{
				Mode:      nicloudsdk.ChatPersonalModelOverrideModeChatDefault,
				Malformed: true,
			},
		},
		{
			name:        "OuterWhitespace",
			raw:         " \tmodel:" + modelConfigID.String() + "\n",
			defaultMode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
			want: chatd.ParsedChatPersonalModelOverride{
				Mode:          nicloudsdk.ChatPersonalModelOverrideModeModel,
				ModelConfigID: modelConfigID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := chatd.ParseChatPersonalModelOverride(tt.raw, tt.defaultMode)
			require.Equal(t, tt.want, got)
		})
	}
}
