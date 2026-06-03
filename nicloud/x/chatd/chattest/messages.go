package chattest

import (
	"encoding/json"

	"github.com/sqlc-dev/pqtype"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// ChatMessageWithParts returns a database chat message whose content is the
// JSON encoding of the provided SDK message parts.
func ChatMessageWithParts(parts []nicloudsdk.ChatMessagePart) database.ChatMessage {
	raw, _ := json.Marshal(parts)
	return database.ChatMessage{
		Content: pqtype.NullRawMessage{RawMessage: raw, Valid: true},
	}
}
