package nicloud

import (
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Get enabled experiments
// @ID get-enabled-experiments
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags General
// @Success 200 {array} nicloudsdk.Experiment
// @Router /api/v2/experiments [get]
func (api *API) handleExperimentsGet(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	httpapi.Write(ctx, rw, http.StatusOK, api.Experiments)
}

// @Summary Get safe experiments
// @ID get-safe-experiments
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags General
// @Success 200 {array} nicloudsdk.Experiment
// @Router /api/v2/experiments/available [get]
func handleExperimentsAvailable(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	httpapi.Write(ctx, rw, http.StatusOK, nicloudsdk.AvailableExperiments{
		Safe: nicloudsdk.ExperimentsSafe,
	})
}
