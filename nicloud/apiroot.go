package nicloud

import (
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary API root handler
// @ID api-root-handler
// @Produce json
// @Tags General
// @Success 200 {object} nicloudsdk.Response
// @Router /api/v2/ [get]
func apiRoot(w http.ResponseWriter, r *http.Request) {
	httpapi.Write(r.Context(), w, http.StatusOK, nicloudsdk.Response{
		//nolint:gocritic
		Message: "👋",
	})
}
