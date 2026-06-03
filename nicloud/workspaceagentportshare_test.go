package nicloud_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbfake"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/provisionersdk/proto"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestPostWorkspaceAgentPortShare(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()
	ownerClient, db := nicloudtest.NewWithDatabase(t, nil)
	owner := nicloudtest.CreateFirstUser(t, ownerClient)
	client, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

	tmpDir := t.TempDir()
	r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
		OrganizationID: owner.OrganizationID,
		OwnerID:        user.ID,
	}).WithAgent(func(agents []*proto.Agent) []*proto.Agent {
		agents[0].Directory = tmpDir
		return agents
	}).Do()
	agents, err := db.GetWorkspaceAgentsInLatestBuildByWorkspaceID(dbauthz.As(ctx, nicloudtest.AuthzUserSubjectWithDB(ctx, t, db, user)), r.Workspace.ID)
	require.NoError(t, err)

	// owner level should fail
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevel("owner"),
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.Error(t, err)

	// invalid level should fail
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevel("invalid"),
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.Error(t, err)

	// invalid protocol should fail
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocol("invalid"),
	})
	require.Error(t, err)

	// invalid port should fail
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       0,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.Error(t, err)
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       90000000,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
	})
	require.Error(t, err)

	// OK, ignoring template max port share level because we are AGPL
	ps, err := client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTPS,
	})
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelPublic, ps.ShareLevel)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareProtocolHTTPS, ps.Protocol)

	// list
	list, err := client.GetWorkspaceAgentPortShares(ctx, r.Workspace.ID)
	require.NoError(t, err)
	require.Len(t, list.Shares, 1)
	require.EqualValues(t, agents[0].Name, list.Shares[0].AgentName)
	require.EqualValues(t, 8080, list.Shares[0].Port)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelPublic, list.Shares[0].ShareLevel)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareProtocolHTTPS, list.Shares[0].Protocol)

	// update share level and protocol
	ps, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelAuthenticated,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelAuthenticated, ps.ShareLevel)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareProtocolHTTP, ps.Protocol)

	// list
	list, err = client.GetWorkspaceAgentPortShares(ctx, r.Workspace.ID)
	require.NoError(t, err)
	require.Len(t, list.Shares, 1)
	require.EqualValues(t, agents[0].Name, list.Shares[0].AgentName)
	require.EqualValues(t, 8080, list.Shares[0].Port)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelAuthenticated, list.Shares[0].ShareLevel)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareProtocolHTTP, list.Shares[0].Protocol)

	// list 2 ordered by port
	ps, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8081,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTPS,
	})
	require.NoError(t, err)
	list, err = client.GetWorkspaceAgentPortShares(ctx, r.Workspace.ID)
	require.NoError(t, err)
	require.Len(t, list.Shares, 2)
	require.EqualValues(t, 8080, list.Shares[0].Port)
	require.EqualValues(t, 8081, list.Shares[1].Port)
}

func TestGetWorkspaceAgentPortShares(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	ownerClient, db := nicloudtest.NewWithDatabase(t, nil)
	owner := nicloudtest.CreateFirstUser(t, ownerClient)
	client, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

	tmpDir := t.TempDir()
	r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
		OrganizationID: owner.OrganizationID,
		OwnerID:        user.ID,
	}).WithAgent(func(agents []*proto.Agent) []*proto.Agent {
		agents[0].Directory = tmpDir
		return agents
	}).Do()
	agents, err := db.GetWorkspaceAgentsInLatestBuildByWorkspaceID(dbauthz.As(ctx, nicloudtest.AuthzUserSubjectWithDB(ctx, t, db, user)), r.Workspace.ID)
	require.NoError(t, err)

	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.NoError(t, err)

	ps, err := client.GetWorkspaceAgentPortShares(ctx, r.Workspace.ID)
	require.NoError(t, err)
	require.Len(t, ps.Shares, 1)
	require.EqualValues(t, agents[0].Name, ps.Shares[0].AgentName)
	require.EqualValues(t, 8080, ps.Shares[0].Port)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelPublic, ps.Shares[0].ShareLevel)
}

func TestDeleteWorkspaceAgentPortShare(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitLong)
	defer cancel()

	ownerClient, db := nicloudtest.NewWithDatabase(t, nil)
	owner := nicloudtest.CreateFirstUser(t, ownerClient)
	client, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID)

	tmpDir := t.TempDir()
	r := dbfake.WorkspaceBuild(t, db, database.WorkspaceTable{
		OrganizationID: owner.OrganizationID,
		OwnerID:        user.ID,
	}).WithAgent(func(agents []*proto.Agent) []*proto.Agent {
		agents[0].Directory = tmpDir
		return agents
	}).Do()
	agents, err := db.GetWorkspaceAgentsInLatestBuildByWorkspaceID(dbauthz.As(ctx, nicloudtest.AuthzUserSubjectWithDB(ctx, t, db, user)), r.Workspace.ID)
	require.NoError(t, err)

	// create
	ps, err := client.UpsertWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  agents[0].Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelPublic, ps.ShareLevel)

	// delete
	err = client.DeleteWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.DeleteWorkspaceAgentPortShareRequest{
		AgentName: agents[0].Name,
		Port:      8080,
	})
	require.NoError(t, err)

	// delete missing
	err = client.DeleteWorkspaceAgentPortShare(ctx, r.Workspace.ID, nicloudsdk.DeleteWorkspaceAgentPortShareRequest{
		AgentName: agents[0].Name,
		Port:      8080,
	})
	require.Error(t, err)

	_, err = db.GetWorkspaceAgentPortShare(dbauthz.As(ctx, nicloudtest.AuthzUserSubjectWithDB(ctx, t, db, user)), database.GetWorkspaceAgentPortShareParams{
		WorkspaceID: r.Workspace.ID,
		AgentName:   agents[0].Name,
		Port:        8080,
	})
	require.Error(t, err)
}
