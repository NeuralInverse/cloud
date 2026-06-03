package httpmw

import (
	"context"
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type notificationTemplateParamContextKey struct{}

// NotificationTemplateParam returns the template from the ExtractNotificationTemplateParam handler.
func NotificationTemplateParam(r *http.Request) database.NotificationTemplate {
	template, ok := r.Context().Value(notificationTemplateParamContextKey{}).(database.NotificationTemplate)
	if !ok {
		panic("developer error: notification template middleware not used")
	}
	return template
}

// ExtractNotificationTemplateParam grabs a notification template from the "notification_template" URL parameter.
func ExtractNotificationTemplateParam(db database.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			notifTemplateID, parsed := ParseUUIDParam(rw, r, "notification_template")
			if !parsed {
				return
			}
			nt, err := db.GetNotificationTemplateByID(r.Context(), notifTemplateID)
			if httpapi.Is404Error(err) {
				httpapi.ResourceNotFound(rw)
				return
			}
			if err != nil {
				httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
					Message: "Internal error fetching notification template.",
					Detail:  err.Error(),
				})
				return
			}

			ctx = context.WithValue(ctx, notificationTemplateParamContextKey{}, nt)
			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}
