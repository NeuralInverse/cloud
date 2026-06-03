package nicloud

import (
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Get deployment config
// @ID get-deployment-config
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags General
// @Success 200 {object} nicloudsdk.DeploymentConfig
// @Router /api/v2/deployment/config [get]
func (api *API) deploymentValues(rw http.ResponseWriter, r *http.Request) {
	if !api.Authorize(r, policy.ActionRead, rbac.ResourceDeploymentConfig) {
		httpapi.Forbidden(rw)
		return
	}

	values, err := api.DeploymentValues.WithoutSecrets()
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	httpapi.Write(
		r.Context(), rw, http.StatusOK,
		nicloudsdk.DeploymentConfig{
			Values:  values,
			Options: api.DeploymentOptions,
		},
	)
}

// @Summary Get deployment stats
// @ID get-deployment-stats
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags General
// @Success 200 {object} nicloudsdk.DeploymentStats
// @Router /api/v2/deployment/stats [get]
func (api *API) deploymentStats(rw http.ResponseWriter, r *http.Request) {
	if !api.Authorize(r, policy.ActionRead, rbac.ResourceDeploymentStats) {
		httpapi.Forbidden(rw)
		return
	}

	stats, ok := api.metricsCache.DeploymentStats()
	if !ok {
		httpapi.Write(r.Context(), rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Deployment stats are still processing!",
		})
		return
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, stats)
}

// @Summary Build info
// @ID build-info
// @Produce json
// @Tags General
// @Success 200 {object} nicloudsdk.BuildInfoResponse
// @Router /api/v2/buildinfo [get]
func buildInfoHandler(resp nicloudsdk.BuildInfoResponse) http.HandlerFunc {
	// This is in a handler so that we can generate API docs info.
	return func(rw http.ResponseWriter, r *http.Request) {
		httpapi.Write(r.Context(), rw, http.StatusOK, resp)
	}
}

// @Summary SSH Config
// @ID ssh-config
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags General
// @Success 200 {object} nicloudsdk.SSHConfigResponse
// @Router /api/v2/deployment/ssh [get]
func (api *API) sshConfig(rw http.ResponseWriter, r *http.Request) {
	httpapi.Write(r.Context(), rw, http.StatusOK, api.SSHConfig)
}
