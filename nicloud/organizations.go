package nicloud

import (
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/db2sdk"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/slice"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Get organizations
// @ID get-organizations
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Organizations
// @Success 200 {object} []nicloudsdk.Organization
// @Router /api/v2/organizations [get]
func (api *API) organizations(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	organizations, err := api.Database.GetOrganizations(ctx, database.GetOrganizationsParams{})
	if httpapi.Is404Error(err) {
		httpapi.ResourceNotFound(rw)
		return
	}
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Internal error fetching organizations.",
			Detail:  err.Error(),
		})
		return
	}

	httpapi.Write(ctx, rw, http.StatusOK, slice.List(organizations, db2sdk.Organization))
}

// @Summary Get organization by ID
// @ID get-organization-by-id
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Organizations
// @Param organization path string true "Organization ID" format(uuid)
// @Success 200 {object} nicloudsdk.Organization
// @Router /api/v2/organizations/{organization} [get]
func (*API) organization(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	organization := httpmw.OrganizationParam(r)

	httpapi.Write(ctx, rw, http.StatusOK, db2sdk.Organization(organization))
}
