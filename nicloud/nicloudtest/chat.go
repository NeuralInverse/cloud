package nicloudtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd"
	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd/chatprovider"
	"github.com/NeuralInverse/cloud/v2/nicloud/x/chatd/chattest"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

const (
	// TestChatProviderOpenAICompat is the default provider for chat runtime tests.
	TestChatProviderOpenAICompat = "openai-compat"
	// TestChatProviderAPIKey is a non-secret API key for local chat providers.
	TestChatProviderAPIKey = "test-api-key"
	// TestChatModelOpenAICompat is the default model for chat runtime tests.
	TestChatModelOpenAICompat = "gpt-4o-mini"
)

// OpenAICompatProviderAPIKeys returns provider keys that route OpenAI-compatible
// chat calls to baseURL.
func OpenAICompatProviderAPIKeys(baseURL string) chatprovider.ProviderAPIKeys {
	return chatprovider.ProviderAPIKeys{
		ByProvider: map[string]string{
			TestChatProviderOpenAICompat: TestChatProviderAPIKey,
		},
		BaseURLByProvider: map[string]string{
			TestChatProviderOpenAICompat: baseURL,
		},
	}
}

// FakeOpenAICompatProviderAPIKeys starts a fake OpenAI-compatible provider and
// returns provider keys for nicloudtest.Options.
func FakeOpenAICompatProviderAPIKeys(t testing.TB) chatprovider.ProviderAPIKeys {
	t.Helper()
	return OpenAICompatProviderAPIKeys(chattest.OpenAI(t))
}

// CreateOpenAICompatChatModelConfig creates the default provider and model
// config used by chat runtime tests. Tests can pass a baseURL to route chat work
// to a specific local provider. If baseURL is empty, this helper starts a fake
// OpenAI-compatible provider.
func CreateOpenAICompatChatModelConfig(
	t testing.TB,
	client *nicloudsdk.ExperimentalClient,
	baseURL string,
) nicloudsdk.ChatModelConfig {
	t.Helper()

	if baseURL == "" {
		baseURL = chattest.OpenAI(t)
	}

	ctx := testutil.Context(t, testutil.WaitLong)
	provider, err := client.CreateAIProvider(ctx, nicloudsdk.CreateAIProviderRequest{
		Type:    nicloudsdk.AIProviderType(TestChatProviderOpenAICompat),
		Name:    "test-" + uuid.NewString(),
		BaseURL: baseURL,
		Enabled: true,
		APIKeys: []string{TestChatProviderAPIKey},
	})
	require.NoError(t, err)
	contextLimit := int64(4096)
	isDefault := true
	modelConfig, err := client.CreateChatModelConfig(ctx, nicloudsdk.CreateChatModelConfigRequest{
		Provider:     TestChatProviderOpenAICompat,
		AIProviderID: &provider.ID,
		Model:        TestChatModelOpenAICompat,
		ContextLimit: &contextLimit,
		IsDefault:    &isDefault,
	})
	require.NoError(t, err)
	return modelConfig
}

// WaitForChatSettled waits for a chat to leave active processing and drains
// tracked chat daemon work before returning the final row.
func WaitForChatSettled(
	ctx context.Context,
	t testing.TB,
	api *nicloud.API,
	chatID uuid.UUID,
) database.Chat {
	t.Helper()

	require.NotNil(t, api)
	waitForChatTerminalState(ctx, t, api.Database, chatID)

	server := api.ChatDaemonForTest()
	require.NotNil(t, server)
	chatd.WaitUntilIdleForTest(server)

	chat, err := getChatByIDAsSystem(ctx, api.Database, chatID)
	require.NoError(t, err)
	return chat
}

func waitForChatTerminalState(
	ctx context.Context,
	t testing.TB,
	db database.Store,
	chatID uuid.UUID,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		chat, err := getChatByIDAsSystem(ctx, db, chatID)
		if err != nil {
			return false
		}
		return chat.Status != database.ChatStatusPending && chat.Status != database.ChatStatusRunning
	}, testutil.WaitLong, testutil.IntervalFast)
}

func getChatByIDAsSystem(
	ctx context.Context,
	db database.Store,
	chatID uuid.UUID,
) (database.Chat, error) {
	// Test helper needs system scope to observe chatd-owned status changes.
	//nolint:gocritic
	return db.GetChatByID(dbauthz.AsSystemRestricted(ctx), chatID)
}
