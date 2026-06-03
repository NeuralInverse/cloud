package taskstatus

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"cdr.dev/slog/v3"
	agentproto "github.com/NeuralInverse/cloud/v2/agent/proto"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/agentsdk"
	"github.com/coder/quartz"
)

// client abstracts the details of using nicloudsdk.Client for workspace operations.
// This interface allows for easier testing by enabling mock implementations and
// provides a cleaner separation of concerns.
//
// The interface is designed to be initialized in two phases:
// 1. Create the client with newClient(niClient)
// 2. Configure logging when the io.Writer is available in Run()
type client interface {
	// CreateUserWorkspace creates a workspace for a user.
	CreateUserWorkspace(ctx context.Context, userID string, req nicloudsdk.CreateWorkspaceRequest) (nicloudsdk.Workspace, error)

	// WorkspaceByOwnerAndName retrieves a workspace by owner and name.
	WorkspaceByOwnerAndName(ctx context.Context, owner string, name string, params nicloudsdk.WorkspaceOptions) (nicloudsdk.Workspace, error)

	// WorkspaceExternalAgentCredentials retrieves credentials for an external agent.
	WorkspaceExternalAgentCredentials(ctx context.Context, workspaceID uuid.UUID, agentName string) (nicloudsdk.ExternalAgentCredentials, error)

	// watchWorkspace watches for updates to a workspace.
	watchWorkspace(ctx context.Context, workspaceID uuid.UUID) (<-chan nicloudsdk.Workspace, error)

	// deleteWorkspace deletes the workspace by creating a build with delete transition.
	deleteWorkspace(ctx context.Context, workspaceID uuid.UUID) error

	// initialize sets up the client with the provided logger, which is only available after Run() is called.
	initialize(logger slog.Logger)
}

// appStatusUpdater abstracts the details of updating app status via the
// Agent dRPC API. This interface is separate from client because it
// requires an agent token which is only available after creating an
// external workspace.
type appStatusUpdater interface {
	// updateAppStatus sends a status update for a workspace app.
	updateAppStatus(ctx context.Context, req *agentproto.UpdateAppStatusRequest) error

	// initialize establishes the dRPC connection using the provided
	// agent token. Must be called before updateAppStatus.
	initialize(ctx context.Context, logger slog.Logger, agentToken string) error

	// close cleanly shuts down the underlying dRPC connection.
	close() error
}

// sdkClient is the concrete implementation of the client interface using
// nicloudsdk.Client.
type sdkClient struct {
	niClient *nicloudsdk.Client
	clock       quartz.Clock
	logger      slog.Logger
}

// newClient creates a new client implementation using the provided nicloudsdk.Client.
func newClient(niClient *nicloudsdk.Client) client {
	return &sdkClient{
		niClient: niClient,
		clock:       quartz.NewReal(),
	}
}

func (c *sdkClient) CreateUserWorkspace(ctx context.Context, userID string, req nicloudsdk.CreateWorkspaceRequest) (nicloudsdk.Workspace, error) {
	return c.niClient.CreateUserWorkspace(ctx, userID, req)
}

func (c *sdkClient) WorkspaceByOwnerAndName(ctx context.Context, owner string, name string, params nicloudsdk.WorkspaceOptions) (nicloudsdk.Workspace, error) {
	return c.niClient.WorkspaceByOwnerAndName(ctx, owner, name, params)
}

func (c *sdkClient) WorkspaceExternalAgentCredentials(ctx context.Context, workspaceID uuid.UUID, agentName string) (nicloudsdk.ExternalAgentCredentials, error) {
	return c.niClient.WorkspaceExternalAgentCredentials(ctx, workspaceID, agentName)
}

func (c *sdkClient) watchWorkspace(ctx context.Context, workspaceID uuid.UUID) (<-chan nicloudsdk.Workspace, error) {
	return c.niClient.WatchWorkspace(ctx, workspaceID)
}

func (c *sdkClient) deleteWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	// Create a build with delete transition to delete the workspace
	_, err := c.niClient.CreateWorkspaceBuild(ctx, workspaceID, nicloudsdk.CreateWorkspaceBuildRequest{
		Transition: nicloudsdk.WorkspaceTransitionDelete,
		Reason:     nicloudsdk.CreateWorkspaceBuildReasonCLI,
	})
	if err != nil {
		return xerrors.Errorf("create delete build: %w", err)
	}
	return nil
}

func (c *sdkClient) initialize(logger slog.Logger) {
	// Configure the neuralinverse client logging
	c.logger = logger
	c.niClient.SetLogger(logger)
	c.niClient.SetLogBodies(true)
}

// sdkAppStatusUpdater is the concrete implementation of the
// appStatusUpdater interface. It dials the Agent dRPC endpoint once
// during initialize and reuses the connection for all subsequent
// UpdateAppStatus calls.
type sdkAppStatusUpdater struct {
	drpcClient agentproto.DRPCAgentClient28
	url        *url.URL
	httpClient *http.Client
}

// newAppStatusUpdater creates a new appStatusUpdater implementation.
func newAppStatusUpdater(client *nicloudsdk.Client) appStatusUpdater {
	return &sdkAppStatusUpdater{
		url:        client.URL,
		httpClient: client.HTTPClient,
	}
}

func (u *sdkAppStatusUpdater) updateAppStatus(ctx context.Context, req *agentproto.UpdateAppStatusRequest) error {
	if u.drpcClient == nil {
		return xerrors.New("dRPC client not initialized - call initialize first")
	}
	_, err := u.drpcClient.UpdateAppStatus(ctx, req)
	return err
}

func (u *sdkAppStatusUpdater) close() error {
	if u.drpcClient == nil {
		return nil
	}
	return u.drpcClient.DRPCConn().Close()
}

func (u *sdkAppStatusUpdater) initialize(ctx context.Context, logger slog.Logger, agentToken string) error {
	agentClient := agentsdk.New(
		u.url,
		agentsdk.WithFixedToken(agentToken),
		nicloudsdk.WithHTTPClient(u.httpClient),
		nicloudsdk.WithLogger(logger),
		nicloudsdk.WithLogBodies(),
	)
	drpcClient, _, err := agentClient.ConnectRPC29WithRole(ctx, "")
	if err != nil {
		return xerrors.Errorf("connect to agent dRPC endpoint: %w", err)
	}
	u.drpcClient = drpcClient
	return nil
}

// Ensure sdkClient implements the client interface.
var _ client = (*sdkClient)(nil)

// Ensure sdkAppStatusUpdater implements the appStatusUpdater interface.
var _ appStatusUpdater = (*sdkAppStatusUpdater)(nil)
