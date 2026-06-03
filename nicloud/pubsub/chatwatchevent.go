package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// ChatWatchEventChannel returns the pubsub channel for chat
// lifecycle events scoped to a single user.
func ChatWatchEventChannel(ownerID uuid.UUID) string {
	return fmt.Sprintf("chat:owner:%s", ownerID)
}

// HandleChatWatchEvent wraps a typed callback for
// ChatWatchEvent messages delivered via pubsub.
func HandleChatWatchEvent(cb func(ctx context.Context, payload nicloudsdk.ChatWatchEvent, err error)) func(ctx context.Context, message []byte, err error) {
	return func(ctx context.Context, message []byte, err error) {
		if err != nil {
			cb(ctx, nicloudsdk.ChatWatchEvent{}, xerrors.Errorf("chat watch event pubsub: %w", err))
			return
		}
		var payload nicloudsdk.ChatWatchEvent
		if err := json.Unmarshal(message, &payload); err != nil {
			cb(ctx, nicloudsdk.ChatWatchEvent{}, xerrors.Errorf("unmarshal chat watch event: %w", err))
			return
		}

		cb(ctx, payload, err)
	}
}
