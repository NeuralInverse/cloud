package chat

import (
	"context"
	"io"

	"github.com/google/uuid"

	"cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type chatClient interface {
	SetLogger(logger slog.Logger)
	SetLogBodies(logBodies bool)
	CreateChat(ctx context.Context, req nicloudsdk.CreateChatRequest) (nicloudsdk.Chat, error)
	StreamChat(ctx context.Context, chatID uuid.UUID, opts *nicloudsdk.StreamChatOptions) (<-chan nicloudsdk.ChatStreamEvent, io.Closer, error)
	CreateChatMessage(ctx context.Context, chatID uuid.UUID, req nicloudsdk.CreateChatMessageRequest) (nicloudsdk.CreateChatMessageResponse, error)
	UpdateChat(ctx context.Context, chatID uuid.UUID, req nicloudsdk.UpdateChatRequest) error
}

type sdkChatClient struct {
	client *nicloudsdk.ExperimentalClient
}

func newChatClient(client *nicloudsdk.Client) chatClient {
	return &sdkChatClient{client: nicloudsdk.NewExperimentalClient(client)}
}

func (c *sdkChatClient) SetLogger(logger slog.Logger) {
	c.client.SetLogger(logger)
}

func (c *sdkChatClient) SetLogBodies(logBodies bool) {
	c.client.SetLogBodies(logBodies)
}

func (c *sdkChatClient) CreateChat(ctx context.Context, req nicloudsdk.CreateChatRequest) (nicloudsdk.Chat, error) {
	return c.client.CreateChat(ctx, req)
}

func (c *sdkChatClient) StreamChat(ctx context.Context, chatID uuid.UUID, opts *nicloudsdk.StreamChatOptions) (<-chan nicloudsdk.ChatStreamEvent, io.Closer, error) {
	return c.client.StreamChat(ctx, chatID, opts)
}

func (c *sdkChatClient) CreateChatMessage(ctx context.Context, chatID uuid.UUID, req nicloudsdk.CreateChatMessageRequest) (nicloudsdk.CreateChatMessageResponse, error) {
	return c.client.CreateChatMessage(ctx, chatID, req)
}

func (c *sdkChatClient) UpdateChat(ctx context.Context, chatID uuid.UUID, req nicloudsdk.UpdateChatRequest) error {
	return c.client.UpdateChat(ctx, chatID, req)
}

var _ chatClient = (*sdkChatClient)(nil)
