package nicloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloud/audit"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac/policy"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Get notifications settings
// @ID get-notifications-settings
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Notifications
// @Success 200 {object} nicloudsdk.NotificationsSettings
// @Router /api/v2/notifications/settings [get]
func (api *API) notificationsSettings(rw http.ResponseWriter, r *http.Request) {
	settingsJSON, err := api.Database.GetNotificationsSettings(r.Context())
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to fetch current notifications settings.",
			Detail:  err.Error(),
		})
		return
	}

	var settings nicloudsdk.NotificationsSettings
	if len(settingsJSON) > 0 {
		err = json.Unmarshal([]byte(settingsJSON), &settings)
		if err != nil {
			httpapi.Write(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.Response{
				Message: "Failed to unmarshal notifications settings.",
				Detail:  err.Error(),
			})
			return
		}
	}
	httpapi.Write(r.Context(), rw, http.StatusOK, settings)
}

// @Summary Update notifications settings
// @ID update-notifications-settings
// @Security Neural Inverse CloudSessionToken
// @Accept json
// @Produce json
// @Tags Notifications
// @Param request body nicloudsdk.NotificationsSettings true "Notifications settings request"
// @Success 200 {object} nicloudsdk.NotificationsSettings
// @Success 304
// @Router /api/v2/notifications/settings [put]
func (api *API) putNotificationsSettings(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var settings nicloudsdk.NotificationsSettings
	if !httpapi.Read(ctx, rw, r, &settings) {
		return
	}

	settingsJSON, err := json.Marshal(&settings)
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to marshal notifications settings.",
			Detail:  err.Error(),
		})
		return
	}

	currentSettingsJSON, err := api.Database.GetNotificationsSettings(ctx)
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to fetch current notifications settings.",
			Detail:  err.Error(),
		})
		return
	}

	if bytes.Equal(settingsJSON, []byte(currentSettingsJSON)) {
		// See: https://www.rfc-editor.org/rfc/rfc7232#section-4.1
		httpapi.Write(ctx, rw, http.StatusNotModified, nil)
		return
	}

	auditor := api.Auditor.Load()
	aReq, commitAudit := audit.InitRequest[database.NotificationsSettings](rw, &audit.RequestParams{
		Audit:   *auditor,
		Log:     api.Logger,
		Request: r,
		Action:  database.AuditActionWrite,
	})
	defer commitAudit()

	aReq.New = database.NotificationsSettings{
		ID:             uuid.New(),
		NotifierPaused: settings.NotifierPaused,
	}

	err = api.Database.UpsertNotificationsSettings(ctx, string(settingsJSON))
	if err != nil {
		if rbac.IsUnauthorizedError(err) {
			httpapi.Forbidden(rw)
			return
		}
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to update notifications settings.",
			Detail:  err.Error(),
		})

		return
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, settings)
}

// notificationTemplatesByKind gets the notification templates by kind
func (api *API) notificationTemplatesByKind(rw http.ResponseWriter, r *http.Request, kind database.NotificationTemplateKind) {
	ctx := r.Context()

	templates, err := api.Database.GetNotificationTemplatesByKind(ctx, kind)
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: fmt.Sprintf("Failed to retrieve %q notifications templates.", kind),
			Detail:  err.Error(),
		})
		return
	}

	out := convertNotificationTemplates(templates)
	httpapi.Write(r.Context(), rw, http.StatusOK, out)
}

// @Summary Get system notification templates
// @ID get-system-notification-templates
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Notifications
// @Success 200 {array} nicloudsdk.NotificationTemplate
// @Failure 500 {object} nicloudsdk.Response "Failed to retrieve 'system' notifications template"
// @Router /api/v2/notifications/templates/system [get]
func (api *API) systemNotificationTemplates(rw http.ResponseWriter, r *http.Request) {
	api.notificationTemplatesByKind(rw, r, database.NotificationTemplateKindSystem)
}

// @Summary Get custom notification templates
// @ID get-custom-notification-templates
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Notifications
// @Success 200 {array} nicloudsdk.NotificationTemplate
// @Failure 500 {object} nicloudsdk.Response "Failed to retrieve 'custom' notifications template"
// @Router /api/v2/notifications/templates/custom [get]
func (api *API) customNotificationTemplates(rw http.ResponseWriter, r *http.Request) {
	api.notificationTemplatesByKind(rw, r, database.NotificationTemplateKindCustom)
}

// @Summary Get notification dispatch methods
// @ID get-notification-dispatch-methods
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Notifications
// @Success 200 {array} nicloudsdk.NotificationMethodsResponse
// @Router /api/v2/notifications/dispatch-methods [get]
func (api *API) notificationDispatchMethods(rw http.ResponseWriter, r *http.Request) {
	var methods []string
	for _, nm := range database.AllNotificationMethodValues() {
		// Skip inbox method as for now this is an implicit delivery target and should not appear
		// anywhere in the Web UI.
		if nm == database.NotificationMethodInbox {
			continue
		}
		methods = append(methods, string(nm))
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, nicloudsdk.NotificationMethodsResponse{
		AvailableNotificationMethods: methods,
		DefaultNotificationMethod:    api.DeploymentValues.Notifications.Method.Value(),
	})
}

// @Summary Send a test notification
// @ID send-a-test-notification
// @Security Neural Inverse CloudSessionToken
// @Tags Notifications
// @Success 200
// @Router /api/v2/notifications/test [post]
func (api *API) postTestNotification(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		key = httpmw.APIKey(r)
	)

	if !api.Authorize(r, policy.ActionUpdate, rbac.ResourceDeploymentConfig) {
		httpapi.Forbidden(rw)
		return
	}

	if _, err := api.NotificationsEnqueuer.EnqueueWithData(
		//nolint:gocritic // We need to be notifier to send the notification.
		dbauthz.AsNotifier(ctx),
		key.UserID,
		notifications.TemplateTestNotification,
		map[string]string{},
		map[string]any{
			// NOTE(DanielleMaywood):
			// When notifications are enqueued, they are checked to be
			// unique within a single day. This means that if we attempt
			// to send two test notifications to the same user on
			// the same day, the enqueuer will prevent us from sending
			// a second one. We are injecting a timestamp to make the
			// notifications appear different enough to circumvent this
			// deduplication logic.
			"timestamp": api.Clock.Now(),
		},
		"send-test-notification",
	); err != nil {
		api.Logger.Error(ctx, "send notification", slog.Error(err))
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to send test notification",
			Detail:  err.Error(),
		})
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

// @Summary Get user notification preferences
// @ID get-user-notification-preferences
// @Security Neural Inverse CloudSessionToken
// @Produce json
// @Tags Notifications
// @Param user path string true "User ID, name, or me"
// @Success 200 {array} nicloudsdk.NotificationPreference
// @Router /api/v2/users/{user}/notifications/preferences [get]
func (api *API) userNotificationPreferences(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx    = r.Context()
		user   = httpmw.UserParam(r)
		logger = api.Logger.Named("notifications.preferences").With(slog.F("user_id", user.ID))
	)

	prefs, err := api.Database.GetUserNotificationPreferences(ctx, user.ID)
	if err != nil {
		logger.Error(ctx, "failed to retrieve preferences", slog.Error(err))

		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to retrieve user notification preferences.",
			Detail:  err.Error(),
		})
		return
	}

	out := convertNotificationPreferences(prefs)
	httpapi.Write(ctx, rw, http.StatusOK, out)
}

// @Summary Update user notification preferences
// @ID update-user-notification-preferences
// @Security Neural Inverse CloudSessionToken
// @Accept json
// @Produce json
// @Tags Notifications
// @Param request body nicloudsdk.UpdateUserNotificationPreferences true "Preferences"
// @Param user path string true "User ID, name, or me"
// @Success 200 {array} nicloudsdk.NotificationPreference
// @Router /api/v2/users/{user}/notifications/preferences [put]
func (api *API) putUserNotificationPreferences(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx    = r.Context()
		user   = httpmw.UserParam(r)
		logger = api.Logger.Named("notifications.preferences").With(slog.F("user_id", user.ID))
	)

	// Parse request.
	var prefs nicloudsdk.UpdateUserNotificationPreferences
	if !httpapi.Read(ctx, rw, r, &prefs) {
		return
	}

	// Build query params.
	input := database.UpdateUserNotificationPreferencesParams{
		UserID:                  user.ID,
		NotificationTemplateIds: make([]uuid.UUID, 0, len(prefs.TemplateDisabledMap)),
		Disableds:               make([]bool, 0, len(prefs.TemplateDisabledMap)),
	}
	for tmplID, disabled := range prefs.TemplateDisabledMap {
		id, err := uuid.Parse(tmplID)
		if err != nil {
			logger.Warn(ctx, "failed to parse notification template UUID", slog.F("input", tmplID), slog.Error(err))

			httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
				Message: "Unable to parse notification template UUID.",
				Detail:  err.Error(),
			})
			return
		}

		input.NotificationTemplateIds = append(input.NotificationTemplateIds, id)
		input.Disableds = append(input.Disableds, disabled)
	}

	// Update preferences with params.
	updated, err := api.Database.UpdateUserNotificationPreferences(ctx, input)
	if err != nil {
		logger.Error(ctx, "failed to update preferences", slog.Error(err))

		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to update user notifications preferences.",
			Detail:  err.Error(),
		})
		return
	}

	// Preferences updated, now fetch all preferences belonging to this user.
	logger.Info(ctx, "updated preferences", slog.F("count", updated))

	userPrefs, err := api.Database.GetUserNotificationPreferences(ctx, user.ID)
	if err != nil {
		logger.Error(ctx, "failed to retrieve preferences", slog.Error(err))

		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to retrieve user notifications preferences.",
			Detail:  err.Error(),
		})
		return
	}

	out := convertNotificationPreferences(userPrefs)
	httpapi.Write(ctx, rw, http.StatusOK, out)
}

// @Summary Send a custom notification
// @ID send-a-custom-notification
// @Security Neural Inverse CloudSessionToken
// @Tags Notifications
// @Accept json
// @Produce json
// @Param request body nicloudsdk.CustomNotificationRequest true "Provide a non-empty title or message"
// @Success 204 "No Content"
// @Failure 400 {object} nicloudsdk.Response "Invalid request body"
// @Failure 403 {object} nicloudsdk.Response "System users cannot send custom notifications"
// @Failure 500 {object} nicloudsdk.Response "Failed to send custom notification"
// @Router /api/v2/notifications/custom [post]
func (api *API) postCustomNotification(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx    = r.Context()
		apiKey = httpmw.APIKey(r)
	)

	// Parse request
	var req nicloudsdk.CustomNotificationRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	// Validate request: require `content` and non-empty `title` and `message`
	if err := req.Validate(); err != nil {
		api.Logger.Error(ctx, "send custom notification: validation failed", slog.Error(err))
		httpapi.Write(ctx, rw, http.StatusBadRequest, nicloudsdk.Response{
			Message: "Invalid request body",
			Detail:  err.Error(),
		})
		return
	}

	// Block system users from sending custom notifications
	user, err := api.Database.GetUserByID(ctx, apiKey.UserID)
	if err != nil {
		api.Logger.Error(ctx, "send custom notification", slog.Error(err))
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to send custom notification",
			Detail:  err.Error(),
		})
		return
	}
	if user.IsSystem {
		api.Logger.Error(ctx, "send custom notification: system user is not allowed",
			slog.F("id", user.ID.String()), slog.F("name", user.Name))
		httpapi.Write(ctx, rw, http.StatusForbidden, nicloudsdk.Response{
			Message: "Forbidden",
			Detail:  "System users cannot send custom notifications.",
		})
		return
	}

	if _, err := api.NotificationsEnqueuer.EnqueueWithData(
		//nolint:gocritic // We need to be notifier to send the notification.
		dbauthz.AsNotifier(ctx),
		user.ID,
		notifications.TemplateCustomNotification,
		map[string]string{
			"custom_title":   req.Content.Title,
			"custom_message": req.Content.Message,
		},
		map[string]any{
			// Current dedupe is done via an hash of (template, user, method, payload, targets, day).
			// Include a minute-bucketed timestamp to bypass per-day dedupe for self-sends,
			// letting the caller resend identical content the same day (but not more than
			// once per minute).
			// TODO(ssncferreira): When custom notifications can target multiple users/roles,
			//   enforce proper deduplication across recipients to reduce noise and prevent spam.
			"dedupe_bypass_ts": api.Clock.Now().UTC().Truncate(time.Minute),
		},
		user.ID.String(),
	); err != nil {
		api.Logger.Error(ctx, "send custom notification", slog.Error(err))
		httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Failed to send custom notification",
			Detail:  err.Error(),
		})
		return
	}

	rw.WriteHeader(http.StatusNoContent)
}

func convertNotificationTemplates(in []database.NotificationTemplate) (out []nicloudsdk.NotificationTemplate) {
	for _, tmpl := range in {
		out = append(out, nicloudsdk.NotificationTemplate{
			ID:               tmpl.ID,
			Name:             tmpl.Name,
			TitleTemplate:    tmpl.TitleTemplate,
			BodyTemplate:     tmpl.BodyTemplate,
			Actions:          string(tmpl.Actions),
			Group:            tmpl.Group.String,
			Method:           string(tmpl.Method.NotificationMethod),
			Kind:             string(tmpl.Kind),
			EnabledByDefault: tmpl.EnabledByDefault,
		})
	}

	return out
}

func convertNotificationPreferences(in []database.NotificationPreference) (out []nicloudsdk.NotificationPreference) {
	for _, pref := range in {
		out = append(out, nicloudsdk.NotificationPreference{
			NotificationTemplateID: pref.NotificationTemplateID,
			Disabled:               pref.Disabled,
			UpdatedAt:              pref.UpdatedAt,
		})
	}

	return out
}
