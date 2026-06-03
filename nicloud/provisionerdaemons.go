package nicloud

import (
	"database/sql"
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/db2sdk"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/sdk2db"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloud/provisionerdserver"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/ptr"
	"github.com/NeuralInverse/cloud/v2/nicloud/util/slice"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Get provisioner daemons
// @ID get-provisioner-daemons
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Provisioning
// @Param organization path string true "Organization ID" format(uuid)
// @Param limit query int false "Page limit"
// @Param ids query []string false "Filter results by job IDs" format(uuid)
// @Param status query nicloudsdk.ProvisionerJobStatus false "Filter results by status" enums(pending,running,succeeded,canceling,canceled,failed)
// @Param tags query object false "Provisioner tags to filter by (JSON of the form {'tag1':'value1','tag2':'value2'})"
// @Success 200 {array} nicloudsdk.ProvisionerDaemon
// @Router /api/v2/organizations/{organization}/provisionerdaemons [get]
func (api *API) provisionerDaemons(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		org = httpmw.OrganizationParam(r)
	)

	// This endpoint returns information about provisioner jobs.
	// For now, only owners and template admins can access provisioner jobs.
	if !api.Authorize(r, policy.ActionRead, rbac.ResourceProvisionerJobs.InOrg(org.ID)) {
		httpapi.ResourceNotFound(rw)
		return
	}

	qp := r.URL.Query()
	p := httpapi.NewQueryParamParser()
	limit := p.PositiveInt32(qp, 50, "limit")
	ids := p.UUIDs(qp, nil, "ids")
	tags := p.JSONStringMap(qp, database.StringMap{}, "tags")
	includeOffline := p.NullableBoolean(qp, sql.NullBool{}, "offline")
	statuses := p.ProvisionerDaemonStatuses(qp, []nicloudsdk.ProvisionerDaemonStatus{}, "status")
	maxAge := p.Duration(qp, 0, "max_age")
	p.ErrorExcessParams(qp)
	if len(p.Errors) > 0 {
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message:     "Invalid query parameters.",
			Validations: p.Errors,
		})
		return
	}

	dbStatuses := sdk2db.ProvisionerDaemonStatuses(statuses)

	daemons, err := api.Database.GetProvisionerDaemonsWithStatusByOrganization(
		ctx,
		database.GetProvisionerDaemonsWithStatusByOrganizationParams{
			OrganizationID:  org.ID,
			StaleIntervalMS: provisionerdserver.StaleInterval.Milliseconds(),
			Limit:           sql.NullInt32{Int32: limit, Valid: limit > 0},
			Offline:         includeOffline,
			Statuses:        dbStatuses,
			MaxAgeMs:        sql.NullInt64{Int64: maxAge.Milliseconds(), Valid: maxAge > 0},
			IDs:             ids,
			Tags:            tags,
		},
	)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Internal error fetching provisioner daemons.",
			Detail:  err.Error(),
		})
		return
	}

	httpapi.Write(ctx, rw, http.StatusOK, slice.List(daemons, func(dbDaemon database.GetProvisionerDaemonsWithStatusByOrganizationRow) nicloudsdk.ProvisionerDaemon {
		pd := db2sdk.ProvisionerDaemon(dbDaemon.ProvisionerDaemon)
		var currentJob, previousJob *nicloudsdk.ProvisionerDaemonJob
		if dbDaemon.CurrentJobID.Valid {
			currentJob = &nicloudsdk.ProvisionerDaemonJob{
				ID:                  dbDaemon.CurrentJobID.UUID,
				Status:              nicloudsdk.ProvisionerJobStatus(dbDaemon.CurrentJobStatus.ProvisionerJobStatus),
				TemplateName:        dbDaemon.CurrentJobTemplateName,
				TemplateIcon:        dbDaemon.CurrentJobTemplateIcon,
				TemplateDisplayName: dbDaemon.CurrentJobTemplateDisplayName,
			}
		}
		if dbDaemon.PreviousJobID.Valid {
			previousJob = &nicloudsdk.ProvisionerDaemonJob{
				ID:                  dbDaemon.PreviousJobID.UUID,
				Status:              nicloudsdk.ProvisionerJobStatus(dbDaemon.PreviousJobStatus.ProvisionerJobStatus),
				TemplateName:        dbDaemon.PreviousJobTemplateName,
				TemplateIcon:        dbDaemon.PreviousJobTemplateIcon,
				TemplateDisplayName: dbDaemon.PreviousJobTemplateDisplayName,
			}
		}

		// Add optional fields.
		pd.KeyName = &dbDaemon.KeyName
		pd.Status = ptr.Ref(nicloudsdk.ProvisionerDaemonStatus(dbDaemon.Status))
		pd.CurrentJob = currentJob
		pd.PreviousJob = previousJob

		return pd
	}))
}
