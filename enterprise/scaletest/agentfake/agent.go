package agentfake

import (
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"cdr.dev/slog/v3"
	"github.com/coder/coder/v2/agent/proto"
	"github.com/coder/coder/v2/codersdk/agentsdk"
	"github.com/coder/quartz"
)

const reconnectBackoff = 1 * time.Second

// Agent is a single fake agent. It owns one workspace-agent auth token and one dRPC connection to coderd.
type Agent struct {
	coderURL *url.URL
	token    string
	logger   slog.Logger

	Clock quartz.Clock

	// Zero ConnectionReportInterval disables fake connection reporting.
	ConnectionReportInterval time.Duration
	ConnectionReportDuration time.Duration

	cancel context.CancelFunc
}

func NewAgent(coderURL *url.URL, token string, logger slog.Logger) *Agent {
	return &Agent{
		coderURL: coderURL,
		token:    token,
		logger:   logger,
	}
}

// Run opens a dRPC websocket to coderd as the "agent" role and keeps it open until ctx is canceled or Close is called.
// On transient failures (e.g., coderd restart, brief auth churn while the workspace build is finalizing) Run reconnects
// with a small backoff.
// Returns nil when ctx is canceled or Close is called, and a non-nil error only if ctx returns a non-context error.
func (a *Agent) Run(ctx context.Context) error {
	// Tie a.closed into ctx so a single select can wait on either.
	runCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	defer a.cancel()

	client := agentsdk.New(a.coderURL, agentsdk.WithFixedToken(a.token))
	for {
		if err := runCtx.Err(); err != nil {
			return nil
		}
		err := a.connectAndServe(runCtx, client)
		if err != nil && runCtx.Err() == nil {
			a.logger.Warn(runCtx, "fake agent dRPC stream ended; reconnecting",
				slog.Error(err))
		}
		select {
		case <-runCtx.Done():
			return nil
		case <-time.After(reconnectBackoff):
		}
	}
}

// connectAndServe opens one dRPC websocket, announces lifecycle = READY, then blocks until ctx is canceled or the
// connection is closed by either side. Returns the underlying error, if any.
func (a *Agent) connectAndServe(ctx context.Context, client *agentsdk.Client) error {
	rpc, _, err := client.ConnectRPC28WithRole(ctx, "agent")
	if err != nil {
		return xerrors.Errorf("connect dRPC: %w", err)
	}
	conn := rpc.DRPCConn()
	defer func() {
		_ = conn.Close()
	}()

	// Real agents transition to READY once their startup script finishes. Fakes have no startup script, so they're
	// "ready" the moment the dRPC stream is open. We send this once per (re)connect because coderd's per-connection
	// lifecycle state is reset each time.
	// Failure here is logged but not treated as fatal: the connection itself is what flips Connected, and a transient
	// failure to update lifecycle shouldn't tear the whole agent down.
	if _, err := rpc.UpdateLifecycle(ctx, &proto.UpdateLifecycleRequest{
		Lifecycle: &proto.Lifecycle{
			State:     proto.Lifecycle_READY,
			ChangedAt: timestamppb.Now(),
		},
	}); err != nil && ctx.Err() == nil {
		a.logger.Warn(ctx, "failed to send lifecycle=READY",
			slog.Error(err))
	}

	go a.runConnectionReports(ctx, rpc)

	select {
	case <-ctx.Done():
		return nil
	case <-conn.Closed():
		return xerrors.New("dRPC connection closed by remote")
	}
}

// runConnectionReports emits a periodic synthetic SSH session
// (CONNECT then DISCONNECT) via ReportConnection. Each session uses one
// connection_id so coderd pairs the two halves onto a single
// connection_log row.
func (a *Agent) runConnectionReports(ctx context.Context, rpc proto.DRPCAgentClient28) {
	if a.ConnectionReportInterval <= 0 {
		return
	}
	clock := a.Clock
	if clock == nil {
		clock = quartz.NewReal()
	}

	// Wake often enough to honor both interval and duration. The select
	// branches gate which action (if any) fires on each tick.
	tick := a.ConnectionReportInterval
	if a.ConnectionReportDuration > 0 && a.ConnectionReportDuration < tick {
		tick = a.ConnectionReportDuration
	}
	ticker := clock.NewTicker(tick, "agentfake", "connectionReports")
	defer ticker.Stop()

	var (
		openID   uuid.UUID
		closeAt  time.Time
		nextOpen = clock.Now().Add(a.ConnectionReportInterval)
	)
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			switch {
			case openID != uuid.Nil && !now.Before(closeAt):
				a.sendConnection(ctx, rpc, openID, proto.Connection_DISCONNECT, now)
				openID = uuid.Nil
				nextOpen = now.Add(a.ConnectionReportInterval)
			case openID == uuid.Nil && !now.Before(nextOpen):
				id := uuid.New()
				closeAt = now.Add(a.ConnectionReportDuration)
				if a.sendConnection(ctx, rpc, id, proto.Connection_CONNECT, now) {
					openID = id
				} else {
					// CONNECT failed; try again next interval. No
					// dangling openID means we won't desync.
					nextOpen = now.Add(a.ConnectionReportInterval)
				}
			}
		}
	}
}

func (a *Agent) sendConnection(ctx context.Context, rpc proto.DRPCAgentClient28, id uuid.UUID, action proto.Connection_Action, now time.Time) bool {
	_, err := rpc.ReportConnection(ctx, &proto.ReportConnectionRequest{
		Connection: &proto.Connection{
			Id:        id[:],
			Action:    action,
			Type:      proto.Connection_SSH,
			Timestamp: timestamppb.New(now),
		},
	})
	if err != nil && ctx.Err() == nil {
		a.logger.Debug(ctx, "report connection failed",
			slog.F("action", action.String()),
			slog.Error(err))
		return false
	}
	return true
}

// Close stops the agent. Safe to call multiple times.
func (a *Agent) Close() {
	if a.cancel != nil {
		a.cancel()
	}
}
