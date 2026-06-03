package nicloud

import (
	"database/sql"
	"errors"
	"net/http"
	"slices"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Upsert workspace agent port share
// @ID upsert-workspace-agent-port-share
// @Security Neural Inverse CloudSessionToken
// @Accept json
// @Produce json
// @Tags PortSharing
// @Param workspace path string true "Workspace ID" format(uuid)
// @Param request body nicloudsdk.UpsertWorkspaceAgentPortShareRequest true "Upsert port sharing level request"
// @Success 200 {object} nicloudsdk.WorkspaceAgentPortShare
// @Router /api/v2/workspaces/{workspace}/port-share [post]
func (api *API) postWorkspaceAgentPortShare(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspace := httpmw.WorkspaceParam(r)
	portSharer := *api.PortSharer.Load()
	var req nicloudsdk.UpsertWorkspaceAgentPortShareRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	if !req.ShareLevel.ValidPortShareLevel() {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Port sharing level not allowed.",
			Validations: []nicloudsdk.ValidationError{
				{
					Field:  "share_level",
					Detail: "Port sharing level not allowed.",
				},
			},
		})
		return
	}

	if req.Port < 9 || req.Port > 65535 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Port must be between 9 and 65535.",
			Validations: []nicloudsdk.ValidationError{
				{
					Field:  "port",
					Detail: "Port must be between 9 and 65535.",
				},
			},
		})
		return
	}
	if !req.Protocol.ValidPortProtocol() {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Port protocol not allowed.",
		})
		return
	}

	template, err := api.Database.GetTemplateByID(ctx, workspace.TemplateID)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	err = portSharer.AuthorizedLevel(template, req.ShareLevel)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: err.Error(),
		})
		return
	}

	agents, err := api.Database.GetWorkspaceAgentsInLatestBuildByWorkspaceID(ctx, workspace.ID)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	found := false
	for _, agent := range agents {
		if agent.Name == req.AgentName {
			found = true
			break
		}
	}
	if !found {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Agent not found.",
		})
		return
	}

	psl, err := api.Database.UpsertWorkspaceAgentPortShare(ctx, database.UpsertWorkspaceAgentPortShareParams{
		WorkspaceID: workspace.ID,
		AgentName:   req.AgentName,
		Port:        req.Port,
		ShareLevel:  database.AppSharingLevel(req.ShareLevel),
		Protocol:    database.PortShareProtocol(req.Protocol),
	})
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	httpapi.Write(ctx, rw, http.StatusOK, convertPortShare(psl))
}

// @Summary Get workspace agent port shares
// @ID get-workspace-agent-port-shares
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags PortSharing
// @Param workspace path string true "Workspace ID" format(uuid)
// @Success 200 {object} nicloudsdk.WorkspaceAgentPortShares
// @Router /api/v2/workspaces/{workspace}/port-share [get]
func (api *API) workspaceAgentPortShares(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspace := httpmw.WorkspaceParam(r)

	shares, err := api.Database.ListWorkspaceAgentPortShares(ctx, workspace.ID)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	httpapi.Write(ctx, rw, http.StatusOK, nicloudsdk.WorkspaceAgentPortShares{
		Shares: convertPortShares(shares),
	})
}

// @Summary Delete workspace agent port share
// @ID delete-workspace-agent-port-share
// @Security Neural Inverse CloudSessionToken
// @Accept json
// @Tags PortSharing
// @Param workspace path string true "Workspace ID" format(uuid)
// @Param request body nicloudsdk.DeleteWorkspaceAgentPortShareRequest true "Delete port sharing level request"
// @Success 200
// @Router /api/v2/workspaces/{workspace}/port-share [delete]
func (api *API) deleteWorkspaceAgentPortShare(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspace := httpmw.WorkspaceParam(r)
	var req nicloudsdk.DeleteWorkspaceAgentPortShareRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	_, err := api.Database.GetWorkspaceAgentPortShare(ctx, database.GetWorkspaceAgentPortShareParams{
		WorkspaceID: workspace.ID,
		AgentName:   req.AgentName,
		Port:        req.Port,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpapi.Write(ctx, rw, http.StatusNotFound, nicloudsdk.Response{
				Message: "Port share not found.",
			})
			return
		}

		httpapi.InternalServerError(rw, err)
		return
	}

	err = api.Database.DeleteWorkspaceAgentPortShare(ctx, database.DeleteWorkspaceAgentPortShareParams{
		WorkspaceID: workspace.ID,
		AgentName:   req.AgentName,
		Port:        req.Port,
	})
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func convertPortShares(shares []database.WorkspaceAgentPortShare) []nicloudsdk.WorkspaceAgentPortShare {
	converted := []nicloudsdk.WorkspaceAgentPortShare{}
	for _, share := range shares {
		converted = append(converted, convertPortShare(share))
	}
	slices.SortFunc(converted, func(i, j nicloudsdk.WorkspaceAgentPortShare) int {
		return (int)(i.Port - j.Port)
	})
	return converted
}

func convertPortShare(share database.WorkspaceAgentPortShare) nicloudsdk.WorkspaceAgentPortShare {
	return nicloudsdk.WorkspaceAgentPortShare{
		WorkspaceID: share.WorkspaceID,
		AgentName:   share.AgentName,
		Port:        share.Port,
		ShareLevel:  nicloudsdk.WorkspaceAgentPortShareLevel(share.ShareLevel),
		Protocol:    nicloudsdk.WorkspaceAgentPortShareProtocol(share.Protocol),
	}
}
