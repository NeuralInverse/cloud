package agentfake_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog/v3"
	"cdr.dev/slog/v3/sloggers/slogtest"
	"github.com/coder/coder/v2/coderd/coderdtest"
	"github.com/coder/coder/v2/coderd/connectionlog"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbfake"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/enterprise/scaletest/agentfake"
	"github.com/coder/coder/v2/testutil"
	"github.com/coder/quartz"
)

// Assert that our fake agent routine establishes the drpc connection and sets its lifecycle status to Ready.
func TestAgent_ConnectsAndReachesReady(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitLong)

	client, db := coderdtest.NewWithDatabase(t, nil)
	user := coderdtest.CreateFirstUser(t, client)

	r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
		OrganizationID: user.OrganizationID,
		OwnerID:        user.UserID,
	}).WithAgent().Do()

	logger := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true}).Leveled(slog.LevelDebug)
	a := agentfake.NewAgent(client.URL, r.AgentToken, logger)
	t.Cleanup(func() { a.Close() })

	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	runErr := make(chan error, 1)
	go func() {
		runErr <- a.Run(runCtx)
	}()

	coderdtest.NewWorkspaceAgentWaiter(t, client, r.Workspace.ID).
		WithContext(ctx).
		Wait()

	require.Eventually(t, func() bool {
		ws, err := client.Workspace(ctx, r.Workspace.ID)
		if err != nil {
			return false
		}
		for _, res := range ws.LatestBuild.Resources {
			for _, agent := range res.Agents {
				if agent.LifecycleState != codersdk.WorkspaceAgentLifecycleReady {
					return false
				}
			}
		}
		return true
	}, testutil.WaitLong, testutil.IntervalFast,
		"agent never reached Lifecycle=ready in workspace %s", r.Workspace.ID)

	// Cancel Run and confirm a clean exit (nil error, not ctx error).
	cancel()
	select {
	case err := <-runErr:
		require.NoError(t, err, "Agent.Run returned unexpected error")
	case <-ctx.Done():
		t.Fatalf("timed out waiting for Agent.Run to return: %v", ctx.Err())
	}

	// Close is idempotent and safe to call after Run returns.
	a.Close()
	a.Close()
}
func TestAgent_ReportsConnections(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitLong)

	connLogger := connectionlog.NewFake()
	client, db := coderdtest.NewWithDatabase(t, &coderdtest.Options{
		ConnectionLogger: connLogger,
	})
	user := coderdtest.CreateFirstUser(t, client)

	r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
		OrganizationID: user.OrganizationID,
		OwnerID:        user.UserID,
	}).WithAgent().Do()

	const (
		interval = 30 * time.Second
		duration = 5 * time.Second
	)

	mClock := quartz.NewMock(t)
	tickerTrap := mClock.Trap().NewTicker("agentfake", "connectionReports")
	defer tickerTrap.Close()

	logger := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true}).Leveled(slog.LevelDebug)
	a := agentfake.NewAgent(client.URL, r.AgentToken, logger)
	a.Clock = mClock
	a.ConnectionReportInterval = interval
	a.ConnectionReportDuration = duration
	t.Cleanup(func() { a.Close() })

	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	runErr := make(chan error, 1)
	go func() {
		runErr <- a.Run(runCtx)
	}()

	coderdtest.NewWorkspaceAgentWaiter(t, client, r.Workspace.ID).
		WithContext(ctx).
		Wait()

	// Wait for the reporting goroutine to register its ticker before
	// advancing the clock, otherwise Advance returns before the goroutine
	// has a timer to fire.
	tickerTrap.MustWait(ctx).MustRelease(ctx)

	// The ticker wakes every min(interval, duration) = 5s. Advance in
	// 5s steps until the goroutine has fired its first CONNECT (at
	// t=interval).
	advanceUntil := func(want int) {
		t.Helper()
		require.Eventually(t, func() bool {
			mClock.Advance(duration).MustWait(ctx)
			return len(connLogger.ConnectionLogs()) >= want
		}, testutil.WaitShort, testutil.IntervalFast,
			"expected %d connection log entries", want)
	}

	advanceUntil(1)

	logs := connLogger.ConnectionLogs()
	require.Len(t, logs, 1)
	require.Equal(t, database.ConnectionTypeSsh, logs[0].Type)
	require.Equal(t, database.ConnectionStatusConnected, logs[0].ConnectionStatus)
	firstConnID := logs[0].ConnectionID.UUID
	require.NotEqual(t, uuid.Nil, firstConnID)

	advanceUntil(2)
	logs = connLogger.ConnectionLogs()
	require.Equal(t, firstConnID, logs[1].ConnectionID.UUID)
	require.Equal(t, database.ConnectionStatusDisconnected, logs[1].ConnectionStatus)

	advanceUntil(3)
	logs = connLogger.ConnectionLogs()
	require.Equal(t, database.ConnectionStatusConnected, logs[2].ConnectionStatus)
	require.NotEqual(t, firstConnID, logs[2].ConnectionID.UUID)

	cancel()
	select {
	case err := <-runErr:
		require.NoError(t, err, "Agent.Run returned unexpected error")
	case <-ctx.Done():
		t.Fatalf("timed out waiting for Agent.Run to return: %v", ctx.Err())
	}
}

func TestAgent_ReportsConnections_Disabled(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitLong)

	connLogger := connectionlog.NewFake()
	client, db := coderdtest.NewWithDatabase(t, &coderdtest.Options{
		ConnectionLogger: connLogger,
	})
	user := coderdtest.CreateFirstUser(t, client)

	r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
		OrganizationID: user.OrganizationID,
		OwnerID:        user.UserID,
	}).WithAgent().Do()

	mClock := quartz.NewMock(t)
	logger := slogtest.Make(t, &slogtest.Options{IgnoreErrors: true}).Leveled(slog.LevelDebug)
	a := agentfake.NewAgent(client.URL, r.AgentToken, logger)
	a.Clock = mClock
	a.ConnectionReportInterval = 0
	t.Cleanup(func() { a.Close() })

	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	runErr := make(chan error, 1)
	go func() {
		runErr <- a.Run(runCtx)
	}()

	coderdtest.NewWorkspaceAgentWaiter(t, client, r.Workspace.ID).
		WithContext(ctx).
		Wait()

	// Disabled means the goroutine never registers a timer, so there's
	// nothing to advance against. Just give wall-clock time for any
	// (buggy) reporting to leak through.
	time.Sleep(testutil.IntervalSlow)

	require.Empty(t, connLogger.ConnectionLogs(),
		"expected no ReportConnection calls when reporting is disabled")

	cancel()
	select {
	case err := <-runErr:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatalf("timed out waiting for Agent.Run to return: %v", ctx.Err())
	}
}

