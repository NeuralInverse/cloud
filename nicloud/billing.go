package nicloud

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"cdr.dev/slog/v3"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloud/notifications"
)

// checkBillingQuota calls the billing service to verify a user can start a workspace.
// Returns nil if allowed, error with user-facing message if blocked.
// Silently allows if billing is not configured (self-hosted).
func (api *API) checkBillingQuota(ctx context.Context, userID string) error {
	billingURL := api.DeploymentValues.BillingURL.String()
	if billingURL == "" {
		return nil // self-hosted, no billing
	}

	internalKey := api.DeploymentValues.BillingInternalKey.String()
	if internalKey == "" {
		return nil // not configured
	}

	url := fmt.Sprintf("%s/api/billing/can-start/%s", billingURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil // fail open
	}
	req.Header.Set("X-Internal-Key", internalKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil // fail open on network error
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil // fail open
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var result struct {
		Allowed bool   `json:"allowed"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	if !result.Allowed {
		msg := result.Message
		if msg == "" {
			msg = "Workspace quota exceeded. Please add a payment method to continue."
		}
		return xerrors.New(msg)
	}

	return nil
}

// billingRedirect generates a signed JWT and redirects the authenticated user
// to the external billing service. Only active when NEURALINVERSE_BILLING_URL is set.
//
// @Summary Redirect to billing service
// @ID billing-redirect
// @Tags Billing
// @Success 307
// @Router /api/v2/billing/redirect [get]
func (api *API) billingRedirect(rw http.ResponseWriter, r *http.Request) {
	billingURL := api.DeploymentValues.BillingURL.String()
	if billingURL == "" {
		http.NotFound(rw, r)
		return
	}

	secret := api.DeploymentValues.BillingJWTSecret.String()
	if secret == "" {
		http.Error(rw, "Billing not configured", http.StatusInternalServerError)
		return
	}

	apiKey := httpmw.APIKey(r)

	// Look up user from the API key's owner
	user, err := api.Database.GetUserByID(r.Context(), apiKey.UserID)
	if err != nil {
		http.Error(rw, "User not found", http.StatusInternalServerError)
		return
	}

	token, err := createBillingToken(user.ID.String(), user.Email, user.Username, secret)
	if err != nil {
		http.Error(rw, "Failed to generate billing token", http.StatusInternalServerError)
		return
	}

	redirectURL := billingURL + "/auth/cloud-redirect?token=" + token
	http.Redirect(rw, r, redirectURL, http.StatusTemporaryRedirect)
}

// modelRedirect generates a signed JWT and redirects the authenticated user
// to the external model gateway. Only active when NEURALINVERSE_MODEL_URL is set.
//
// @Summary Redirect to model gateway
// @ID model-redirect
// @Tags Model
// @Success 307
// @Router /api/v2/model/redirect [get]
func (api *API) modelRedirect(rw http.ResponseWriter, r *http.Request) {
	modelURL := api.DeploymentValues.ModelURL.String()
	if modelURL == "" {
		http.NotFound(rw, r)
		return
	}

	secret := api.DeploymentValues.ModelJWTSecret.String()
	if secret == "" {
		http.Error(rw, "Model gateway not configured", http.StatusInternalServerError)
		return
	}

	apiKey := httpmw.APIKey(r)

	// Look up user from the API key's owner
	user, err := api.Database.GetUserByID(r.Context(), apiKey.UserID)
	if err != nil {
		http.Error(rw, "User not found", http.StatusInternalServerError)
		return
	}

	token, err := createServiceToken(user.ID.String(), user.Email, user.Username, secret, "model")
	if err != nil {
		http.Error(rw, "Failed to generate model token", http.StatusInternalServerError)
		return
	}

	redirectURL := modelURL + "/auth/cloud-redirect?token=" + token
	http.Redirect(rw, r, redirectURL, http.StatusTemporaryRedirect)
}

func createBillingToken(userID, email, username, secret string) (string, error) {
	return createServiceToken(userID, email, username, secret, "billing")
}

// hardwareRedirect generates a signed JWT and redirects the authenticated user
// to the external hardware gateway. Only active when NEURALINVERSE_HARDWARE_URL is set.
//
// @Summary Redirect to hardware gateway
// @ID hardware-redirect
// @Tags Hardware
// @Success 307
// @Router /api/v2/hardware/redirect [get]
func (api *API) hardwareRedirect(rw http.ResponseWriter, r *http.Request) {
	hardwareURL := api.DeploymentValues.HardwareURL.String()
	if hardwareURL == "" {
		http.NotFound(rw, r)
		return
	}

	secret := api.DeploymentValues.HardwareJWTSecret.String()
	if secret == "" {
		http.Error(rw, "Hardware gateway not configured", http.StatusInternalServerError)
		return
	}

	apiKey := httpmw.APIKey(r)

	// Look up user from the API key's owner
	user, err := api.Database.GetUserByID(r.Context(), apiKey.UserID)
	if err != nil {
		http.Error(rw, "User not found", http.StatusInternalServerError)
		return
	}

	token, err := createServiceToken(user.ID.String(), user.Email, user.Username, secret, "hardware")
	if err != nil {
		http.Error(rw, "Failed to generate hardware token", http.StatusInternalServerError)
		return
	}

	redirectURL := hardwareURL + "/auth/cloud-redirect?token=" + token
	http.Redirect(rw, r, redirectURL, http.StatusTemporaryRedirect)
}

// baseRedirect generates a signed JWT and redirects the authenticated user
// to base.neuralinverse.com. Only active when NEURALINVERSE_BASE_URL is set.
//
// @Summary Redirect to Base (code storage)
// @ID base-redirect
// @Tags Base
// @Success 307
// @Router /api/v2/base/redirect [get]
func (api *API) baseRedirect(rw http.ResponseWriter, r *http.Request) {
	baseURL := api.DeploymentValues.BaseURL.String()
	if baseURL == "" {
		http.NotFound(rw, r)
		return
	}

	secret := api.DeploymentValues.BaseJWTSecret.String()
	apiKey := httpmw.APIKey(r)
	user, err := api.Database.GetUserByID(r.Context(), apiKey.UserID)
	if err != nil {
		http.Error(rw, "User not found", http.StatusInternalServerError)
		return
	}

	if secret != "" {
		token, err := createServiceToken(user.ID.String(), user.Email, user.Username, secret, "base")
		if err != nil {
			http.Error(rw, "Failed to generate base token", http.StatusInternalServerError)
			return
		}
		http.Redirect(rw, r, baseURL+"/auth/cloud-redirect?token="+token, http.StatusTemporaryRedirect)
		return
	}

	// No JWT secret configured — redirect directly (user will be prompted to log in via OAuth2)
	http.Redirect(rw, r, baseURL, http.StatusTemporaryRedirect)
}

func createServiceToken(userID, email, username, secret, service string) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := map[string]interface{}{
		"userId":   userID,
		"email":    email,
		"username": username,
		"service":  service,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(5 * time.Minute).Unix(),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadEnc := base64.RawURLEncoding.EncodeToString(payloadBytes)

	sigInput := header + "." + payloadEnc
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return sigInput + "." + sig, nil
}

// postBillingNotify is an internal endpoint called by the billing service to push
// a quota-exceeded notification into a user's inbox. Only active when
// NEURALINVERSE_BILLING_INTERNAL_KEY is configured — no-op on self-hosted instances.
//
// @Summary Notify user of quota exceeded (internal)
// @ID billing-notify
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body billingNotifyRequest true "Notification request"
// @Success 204
// @Router /api/v2/billing/notify [post]
func (api *API) postBillingNotify(rw http.ResponseWriter, r *http.Request) {
	internalKey := api.DeploymentValues.BillingInternalKey.String()
	if internalKey == "" {
		http.NotFound(rw, r)
		return
	}
	if r.Header.Get("X-Internal-Key") != internalKey {
		http.Error(rw, "Forbidden", http.StatusForbidden)
		return
	}

	var req billingNotifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, "Bad request", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(rw, "Invalid user_id", http.StatusBadRequest)
		return
	}
	workspaceID, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		http.Error(rw, "Invalid workspace_id", http.StatusBadRequest)
		return
	}

	_, err = api.NotificationsEnqueuer.Enqueue(
		r.Context(),
		userID,
		notifications.TemplateWorkspaceQuotaExceeded,
		map[string]string{
			"name":        req.WorkspaceName,
			"billing_url": req.BillingURL,
			"reason":      req.Reason,
		},
		"billing",
		workspaceID, userID,
	)
	if err != nil {
		// Log but don't fail — notification is best-effort
		api.Logger.Warn(r.Context(), "failed to enqueue quota exceeded notification", slog.Error(err))
	}

	rw.WriteHeader(http.StatusNoContent)
}

type billingNotifyRequest struct {
	UserID        string `json:"user_id"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
	BillingURL    string `json:"billing_url"`
	Reason        string `json:"reason"`
}
