package nicloud

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/NeuralInverse/cloud/v2/apiversion"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/tailnet/proto"
	"github.com/coder/websocket"
)

// @Summary Workspace Proxy Coordinate
// @ID workspace-proxy-coordinate
// @Security Neural Inverse CloudSessionToken
// @Tags Enterprise
// @Success 101
// @Router /api/v2/workspaceproxies/me/coordinate [get]
// @x-apidocgen {"skip": true}
func (api *API) workspaceProxyCoordinate(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	version := "1.0"
	msgType := websocket.MessageText
	qv := r.URL.Query().Get("version")
	if qv != "" {
		version = qv
	}
	if err := proto.CurrentVersion.Validate(version); err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Unknown or unsupported API version",
			Validations: []nicloudsdk.ValidationError{
				{Field: "version", Detail: err.Error()},
			},
		})
		return
	}
	maj, _, _ := apiversion.Parse(version)
	if maj >= 2 {
		// Versions 2+ use dRPC over a binary connection
		msgType = websocket.MessageBinary
	}

	api.AGPL.WebsocketWaitMutex.Lock()
	api.AGPL.WebsocketWaitGroup.Add(1)
	api.AGPL.WebsocketWaitMutex.Unlock()
	defer api.AGPL.WebsocketWaitGroup.Done()

	conn, err := websocket.Accept(rw, r, nil)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Failed to accept websocket.",
			Detail:  err.Error(),
		})
		return
	}

	ctx, nc := nicloudsdk.WebsocketNetConn(ctx, conn, msgType)
	defer nc.Close()

	id := uuid.New()
	err = api.tailnetService.ServeMultiAgentClient(ctx, version, nc, id)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, err.Error())
	} else {
		_ = conn.Close(websocket.StatusGoingAway, "")
	}
}
