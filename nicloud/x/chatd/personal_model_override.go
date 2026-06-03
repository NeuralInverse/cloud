package chatd

import (
	"strings"

	"github.com/google/uuid"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// ChatPersonalModelOverrideKeyPrefix is the user config key prefix for
// chat personal model overrides. Values under this prefix should be parsed
// with ParseChatPersonalModelOverride so malformed values use one fallback.
const ChatPersonalModelOverrideKeyPrefix = "chat_personal_model_override:"

// ChatPersonalModelOverrideKey returns the user config key for a chat
// personal model override context. Values stored at the returned key should
// use ParseChatPersonalModelOverride so malformed values fall back safely.
func ChatPersonalModelOverrideKey(
	overrideContext nicloudsdk.ChatPersonalModelOverrideContext,
) string {
	return ChatPersonalModelOverrideKeyPrefix + string(overrideContext)
}

// ParsedChatPersonalModelOverride is a parsed personal model override value.
// When Malformed is true, Mode is the provided default and ModelConfigID is
// uuid.Nil.
type ParsedChatPersonalModelOverride struct {
	Mode          nicloudsdk.ChatPersonalModelOverrideMode
	ModelConfigID uuid.UUID
	Malformed     bool
}

// ParseChatPersonalModelOverride parses a stored personal model override.
// Empty values return defaultMode without marking the value malformed.
// Malformed values return defaultMode, uuid.Nil, and Malformed true.
func ParseChatPersonalModelOverride(
	raw string,
	defaultMode nicloudsdk.ChatPersonalModelOverrideMode,
) ParsedChatPersonalModelOverride {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ParsedChatPersonalModelOverride{Mode: defaultMode}
	}

	switch trimmed {
	case string(nicloudsdk.ChatPersonalModelOverrideModeChatDefault):
		return ParsedChatPersonalModelOverride{
			Mode: nicloudsdk.ChatPersonalModelOverrideModeChatDefault,
		}
	case string(nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault):
		return ParsedChatPersonalModelOverride{
			Mode: nicloudsdk.ChatPersonalModelOverrideModeDeploymentDefault,
		}
	}

	mode, rawModelConfigID, ok := strings.Cut(trimmed, ":")
	if !ok || mode != string(nicloudsdk.ChatPersonalModelOverrideModeModel) {
		return ParsedChatPersonalModelOverride{
			Mode:      defaultMode,
			Malformed: true,
		}
	}
	modelConfigID, err := uuid.Parse(rawModelConfigID)
	if err != nil {
		return ParsedChatPersonalModelOverride{
			Mode:      defaultMode,
			Malformed: true,
		}
	}
	return ParsedChatPersonalModelOverride{
		Mode:          nicloudsdk.ChatPersonalModelOverrideModeModel,
		ModelConfigID: modelConfigID,
	}
}
