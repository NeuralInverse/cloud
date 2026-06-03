package chatdebug_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd/chatdebug"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// toStrings converts a typed string slice to []string for comparison.
func toStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// TestTypesMatchSDK verifies that every chatdebug constant has a
// corresponding nicloudsdk constant with the same string value.
// If this test fails you probably added a constant to one package
// but forgot to update the other.
func TestTypesMatchSDK(t *testing.T) {
	t.Parallel()

	t.Run("RunKind", func(t *testing.T) {
		t.Parallel()
		require.ElementsMatch(t,
			toStrings(chatdebug.AllRunKinds),
			toStrings(nicloudsdk.AllChatDebugRunKinds),
			"chatdebug.AllRunKinds and nicloudsdk.AllChatDebugRunKinds have diverged",
		)
	})

	t.Run("Status", func(t *testing.T) {
		t.Parallel()
		require.ElementsMatch(t,
			toStrings(chatdebug.AllStatuses),
			toStrings(nicloudsdk.AllChatDebugStatuses),
			"chatdebug.AllStatuses and nicloudsdk.AllChatDebugStatuses have diverged",
		)
	})

	t.Run("Operation", func(t *testing.T) {
		t.Parallel()
		require.ElementsMatch(t,
			toStrings(chatdebug.AllOperations),
			toStrings(nicloudsdk.AllChatDebugStepOperations),
			"chatdebug.AllOperations and nicloudsdk.AllChatDebugStepOperations have diverged",
		)
	})
}
