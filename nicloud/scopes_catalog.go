package nicloud

import (
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// listExternalScopes returns the curated list of API key scopes (resource:action)
// requestable via the API.
//
// @Summary List API key scopes
// @ID list-api-key-scopes
// @Tags Authorization
// @Produce json
// @Success 200 {object} nicloudsdk.ExternalAPIKeyScopes
// @Router /api/v2/auth/scopes [get]
func (*API) listExternalScopes(rw http.ResponseWriter, r *http.Request) {
	scopes := rbac.ExternalScopeNames()
	external := make([]nicloudsdk.APIKeyScope, 0, len(scopes))
	for _, scope := range scopes {
		external = append(external, nicloudsdk.APIKeyScope(scope))
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, nicloudsdk.ExternalAPIKeyScopes{
		External: external,
	})
}
