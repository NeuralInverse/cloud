package tunneler_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/NeuralInverse/cloud/v2/agent/agenttest"
	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/workspacesdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/workspacesdk/tunneler"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

// TestTunneler_Integration is an integration test using nicloudtest. It should be removed when we integrate the Tunneler
// into neuralinverse ssh and those integration test cover this functionality.
func TestTunneler_Integration(t *testing.T) {
	t.Parallel()
	ctx := testutil.Context(t, testutil.WaitLong)
	client, store := nicloudtest.NewWithDatabase(t, nil)
	logger := testutil.Logger(t)
	client.SetLogger(logger.Named("client"))
	first := nicloudtest.CreateFirstUser(t, client)
	userClient, user := nicloudtest.CreateAnotherUserMutators(t, client, first.OrganizationID, nil, func(r *nicloudsdk.CreateUserRequestWithOrgs) {
		r.Username = "myuser"
	})
	userClient.SetLogger(logger.Named("userclient"))
	r := dbfake.WorkspaceBuild(t, store, database.WorkspaceTable{
		Name:           "myworkspace",
		OrganizationID: first.OrganizationID,
		OwnerID:        user.ID,
	}).WithAgent().Do()
	wsSDKClient := workspacesdk.New(userClient)
	logs := &bytes.Buffer{}

	app := &sshApplication{
		t:    t,
		ctx:  ctx,
		done: make(chan struct{}),
	}

	tun := tunneler.NewTunneler(wsSDKClient, tunneler.Config{
		WorkspaceID:      r.Workspace.ID,
		App:              app,
		WorkspaceStarter: nil,
		AgentName:        "",
		LogWriter:        logs,
		DebugLogger:      logger.Named("tunneler"),
	})

	testAgent := agenttest.New(t, client.URL, r.AgentToken)
	defer testAgent.Close()

	testutil.TryReceive(ctx, t, app.done)
	// TrimSpace removes line endings, which vary by OS and are not important to this test.
	require.Equal(t, "foo", strings.TrimSpace(app.result))

	err := tun.GracefulShutdown(ctx)
	require.NoError(t, err)
}

type sshApplication struct {
	t      *testing.T
	ctx    context.Context
	client *ssh.Client
	done   chan struct{}
	result string
}

func (s *sshApplication) Close() error {
	return s.client.Close()
}

func (s *sshApplication) Start(conn workspacesdk.AgentConn) error {
	var err error
	s.client, err = conn.SSHClient(s.ctx)
	if err != nil {
		s.t.Error(err)
		return err
	}
	go func() {
		defer close(s.done)
		sess, err := s.client.NewSession()
		if err != nil {
			s.t.Error("failed to create session", err)
		}
		defer sess.Close()
		out, err := sess.Output("echo foo")
		if err != nil {
			s.t.Error("failed to echo", err)
		}
		s.result = string(out)
	}()
	return nil
}
