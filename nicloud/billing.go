package nicloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
)

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

func createBillingToken(userID, email, username, secret string) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payload := map[string]interface{}{
		"userId":   userID,
		"email":    email,
		"username": username,
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
