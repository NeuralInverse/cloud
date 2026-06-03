package nicloud_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/nicloudtest"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/nicloudenttest"
	"github.com/NeuralInverse/cloud/v2/enterprise/nicloud/license"
	"github.com/NeuralInverse/cloud/v2/testutil"
)

func TestWorkspacePortSharePublic(t *testing.T) {
	t.Parallel()

	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{IncludeProvisionerDaemon: true},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{nicloudsdk.FeatureControlSharedPorts: 1},
		},
	})
	client, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
	r := setupWorkspaceAgent(t, client, nicloudsdk.CreateFirstUserResponse{
		UserID:         user.ID,
		OrganizationID: owner.OrganizationID,
	}, 0)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	defer cancel()

	templ, err := client.Template(ctx, r.workspace.TemplateID)
	require.NoError(t, err)
	require.Equal(t, templ.MaxPortShareLevel, nicloudsdk.WorkspaceAgentPortShareLevelOwner)

	// Try to update port share with template max port share level owner.
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  r.sdkAgent.Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.Error(t, err, "Port sharing level not allowed")

	// Update the template max port share level to public
	client.UpdateTemplateMeta(ctx, r.workspace.TemplateID, nicloudsdk.UpdateTemplateMeta{
		MaxPortShareLevel: ptr.Ref(nicloudsdk.WorkspaceAgentPortShareLevelPublic),
	})

	// OK
	ps, err := client.UpsertWorkspaceAgentPortShare(ctx, r.workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  r.sdkAgent.Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelPublic, ps.ShareLevel)
}

func TestWorkspacePortShareOrganization(t *testing.T) {
	t.Parallel()

	ownerClient, owner := nicloudenttest.New(t, &nicloudenttest.Options{
		Options: &nicloudtest.Options{IncludeProvisionerDaemon: true},
		LicenseOptions: &nicloudenttest.LicenseOptions{
			Features: license.Features{nicloudsdk.FeatureControlSharedPorts: 1},
		},
	})
	client, user := nicloudtest.CreateAnotherUser(t, ownerClient, owner.OrganizationID, rbac.RoleTemplateAdmin())
	r := setupWorkspaceAgent(t, client, nicloudsdk.CreateFirstUserResponse{
		UserID:         user.ID,
		OrganizationID: owner.OrganizationID,
	}, 0)
	ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
	defer cancel()

	templ, err := client.Template(ctx, r.workspace.TemplateID)
	require.NoError(t, err)
	require.Equal(t, templ.MaxPortShareLevel, nicloudsdk.WorkspaceAgentPortShareLevelOwner)

	// Try to update port share with template max port share level owner
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  r.sdkAgent.Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelOrganization,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.Error(t, err, "Port sharing level not allowed")

	// Update the template max port share level to organization
	client.UpdateTemplateMeta(ctx, r.workspace.TemplateID, nicloudsdk.UpdateTemplateMeta{
		MaxPortShareLevel: ptr.Ref(nicloudsdk.WorkspaceAgentPortShareLevelOrganization),
	})

	// Try to share a port publicly with template max port share level organization
	_, err = client.UpsertWorkspaceAgentPortShare(ctx, r.workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  r.sdkAgent.Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelPublic,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.Error(t, err, "Port sharing level not allowed")

	// OK
	ps, err := client.UpsertWorkspaceAgentPortShare(ctx, r.workspace.ID, nicloudsdk.UpsertWorkspaceAgentPortShareRequest{
		AgentName:  r.sdkAgent.Name,
		Port:       8080,
		ShareLevel: nicloudsdk.WorkspaceAgentPortShareLevelOrganization,
		Protocol:   nicloudsdk.WorkspaceAgentPortShareProtocolHTTP,
	})
	require.NoError(t, err)
	require.EqualValues(t, nicloudsdk.WorkspaceAgentPortShareLevelOrganization, ps.ShareLevel)
}
