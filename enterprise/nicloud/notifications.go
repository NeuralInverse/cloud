package nicloud

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Update notification template dispatch method
// @ID update-notification-template-dispatch-method
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Param notification_template path string true "Notification template UUID"
// @Tags Enterprise
// @Success 200 "Success"
// @Success 304 "Not modified"
// @Router /api/v2/notifications/templates/{notification_template}/method [put]
func (api *API) updateNotificationTemplateMethod(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx               = r.Context()
		template          = httpmw.NotificationTemplateParam(r)
		auditor           = api.AGPL.Auditor.Load()
		aReq, commitAudit = audit.InitRequest[database.NotificationTemplate](rw, &audit.RequestParams{
			Audit:   *auditor,
			Log:     api.Logger,
			Request: r,
			Action:  database.AuditActionWrite,
		})
	)

	var req nicloudsdk.UpdateNotificationTemplateMethod
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	var nm database.NullNotificationMethod
	if err := nm.Scan(req.Method); err != nil || !nm.Valid || !nm.NotificationMethod.Valid() {
		vals := database.AllNotificationMethodValues()
		acceptable := make([]string, len(vals))
		for i, v := range vals {
			acceptable[i] = string(v)
		}

		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Invalid request to update notification template method",
			Validations: []nicloudsdk.ValidationError{
				{
					Field: "method",
					Detail: fmt.Sprintf("%q is not a valid method; %s are the available options",
						req.Method, strings.Join(acceptable, ", "),
					),
				},
			},
		})
		return
	}

	if template.Method == nm {
		httpapi.Write(ctx, rw, http.StatusNotModified, nicloudsdk.Response{
			Message: "Notification template method unchanged.",
		})
		return
	}

	defer commitAudit()
	aReq.Old = template

	err := api.Database.InTx(func(tx database.Store) error {
		var err error
		template, err = tx.UpdateNotificationTemplateMethodByID(r.Context(), database.UpdateNotificationTemplateMethodByIDParams{
			ID:     template.ID,
			Method: nm,
		})
		if err != nil {
			return xerrors.Errorf("failed to update notification template ID: %w", err)
		}

		return err
	}, nil)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	aReq.New = template

	httpapi.Write(ctx, rw, http.StatusOK, nicloudsdk.Response{
		Message: "Successfully updated notification template method.",
	})
}
