package oauth2provider

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	htmltemplate "html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/justinas/nosurf"
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbtime"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/site"
)

type authorizeParams struct {
	clientID            string
	redirectURL         *url.URL
	redirectURIProvided bool
	responseType        nicloudsdk.OAuth2ProviderResponseType
	scope               []string
	state               string
	resource            string // RFC 8707 resource indicator
	codeChallenge       string // PKCE code challenge
	codeChallengeMethod string // PKCE challenge method
}

func extractAuthorizeParams(r *http.Request, callbackURL *url.URL) (authorizeParams, []nicloudsdk.ValidationError, error) {
	p := httpapi.NewQueryParamParser()
	vals := r.URL.Query()

	// response_type and client_id are always required.
	p.RequiredNotEmpty("response_type", "client_id")

	params := authorizeParams{
		clientID:            p.String(vals, "", "client_id"),
		redirectURL:         p.RedirectURL(vals, callbackURL, "redirect_uri"),
		redirectURIProvided: vals.Get("redirect_uri") != "",
		responseType:        httpapi.ParseCustom(p, vals, "", "response_type", httpapi.ParseEnum[nicloudsdk.OAuth2ProviderResponseType]),
		scope:               strings.Fields(strings.TrimSpace(p.String(vals, "", "scope"))),
		state:               p.String(vals, "", "state"),
		resource:            p.String(vals, "", "resource"),
		codeChallenge:       p.String(vals, "", "code_challenge"),
		codeChallengeMethod: p.String(vals, "", "code_challenge_method"),
	}

	// PKCE is recommended but not required — confidential server-side clients (e.g. Gitea)
	// use client_secret for security instead of code_challenge.

	// Validate resource indicator syntax (RFC 8707): must be absolute URI without fragment
	if err := validateResourceParameter(params.resource); err != nil {
		p.Errors = append(p.Errors, nicloudsdk.ValidationError{
			Field:  "resource",
			Detail: "must be an absolute URI without fragment",
		})
	}

	p.ErrorExcessParams(vals)
	if len(p.Errors) > 0 {
		// Create a readable error message with validation details
		var errorDetails []string
		for _, err := range p.Errors {
			errorDetails = append(errorDetails, err.Error())
		}
		errorMsg := "Invalid query params: " + strings.Join(errorDetails, ", ")
		return authorizeParams{}, p.Errors, xerrors.Errorf(errorMsg)
	}
	return params, nil, nil
}

// isTrustedApp returns true for first-party Neural Inverse apps that should skip the consent screen.
func isTrustedApp(callbackURL string) bool {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return strings.HasSuffix(host, ".neuralinverse.com") || host == "neuralinverse.com"
}

// ShowAuthorizePage handles GET /oauth2/authorize requests to display the HTML authorization page.
// For first-party Neural Inverse apps the consent screen is skipped and the code is issued immediately.
func ShowAuthorizePage(accessURL *url.URL, db database.Store) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		app := httpmw.OAuth2ProviderApp(r)
		ua := httpmw.UserAuthorization(r.Context())

		callbackURL, err := url.Parse(app.CallbackURL)
		if err != nil {
			site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
				Status:      http.StatusInternalServerError,
				HideStatus:  false,
				Title:       "Internal Server Error",
				Description: err.Error(),
				Actions: []site.Action{
					{
						URL:  accessURL.String(),
						Text: "Back to site",
					},
				},
			})
			return
		}

		params, validationErrs, err := extractAuthorizeParams(r, callbackURL)
		if err != nil {
			errStr := make([]string, len(validationErrs))
			for i, err := range validationErrs {
				errStr[i] = err.Detail
			}
			site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
				Status:      http.StatusBadRequest,
				HideStatus:  false,
				Title:       "Invalid Query Parameters",
				Description: "One or more query parameters are missing or invalid.",
				Warnings:    errStr,
				Actions: []site.Action{
					{
						URL:  accessURL.String(),
						Text: "Back to site",
					},
				},
			})
			return
		}

		if params.responseType != nicloudsdk.OAuth2ProviderResponseTypeCode {
			site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
				Status:      http.StatusBadRequest,
				HideStatus:  false,
				Title:       "Unsupported Response Type",
				Description: "Only response_type=code is supported.",
				Actions: []site.Action{
					{
						URL:  accessURL.String(),
						Text: "Back to site",
					},
				},
			})
			return
		}

		cancel := params.redirectURL
		cancelQuery := params.redirectURL.Query()
		cancelQuery.Add("error", "access_denied")
		cancelQuery.Add("error_description", "The resource owner or authorization server denied the request")
		if params.state != "" {
			cancelQuery.Add("state", params.state)
		}
		cancel.RawQuery = cancelQuery.Encode()

		cancelURI := cancel.String()
		if err := nicloudsdk.ValidateRedirectURIScheme(cancel); err != nil {
			site.RenderStaticErrorPage(rw, r, site.ErrorPageData{
				Status:      http.StatusBadRequest,
				HideStatus:  false,
				Title:       "Invalid Callback URL",
				Description: "The application's registered callback URL has an invalid scheme.",
				Actions: []site.Action{
					{
						URL:  accessURL.String(),
						Text: "Back to site",
					},
				},
			})
			return
		}

		// First-party Neural Inverse apps (e.g. base.neuralinverse.com) skip the consent screen.
		if isTrustedApp(app.CallbackURL) {
			apiKey := httpmw.APIKey(r)
			issueAuthCode(rw, r, db, app.ID, apiKey.UserID, params)
			return
		}

		site.RenderOAuthAllowPage(rw, r, site.RenderOAuthAllowData{
			AppIcon: app.Icon,
			AppName: app.Name,
			// #nosec G203 -- The scheme is validated by
			// nicloudsdk.ValidateRedirectURIScheme above.
			CancelURI:    htmltemplate.URL(cancelURI),
			DashboardURL: accessURL.String(),
			CSRFToken:    nosurf.Token(r),
			Username:     ua.FriendlyName,
		})
	}
}

// ProcessAuthorize handles POST /oauth2/authorize requests to process the user's authorization decision
// and generate an authorization code.
func ProcessAuthorize(db database.Store) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		apiKey := httpmw.APIKey(r)
		app := httpmw.OAuth2ProviderApp(r)

		callbackURL, err := url.Parse(app.CallbackURL)
		if err != nil {
			httpapi.WriteOAuth2Error(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.OAuth2ErrorCodeServerError, "Failed to validate query parameters")
			return
		}

		params, _, err := extractAuthorizeParams(r, callbackURL)
		if err != nil {
			httpapi.WriteOAuth2Error(ctx, rw, http.StatusBadRequest, nicloudsdk.OAuth2ErrorCodeInvalidRequest, err.Error())
			return
		}

		// OAuth 2.1 removes the implicit grant. Only
		// authorization code flow is supported.
		if params.responseType != nicloudsdk.OAuth2ProviderResponseTypeCode {
			httpapi.WriteOAuth2Error(ctx, rw, http.StatusBadRequest,
				nicloudsdk.OAuth2ErrorCodeUnsupportedResponseType,
				"Only response_type=code is supported")
			return
		}

		// code_challenge is required (enforced by RequiredNotEmpty above),
		// but default the method to S256 if omitted.
		if params.codeChallengeMethod == "" {
			params.codeChallengeMethod = string(nicloudsdk.OAuth2PKCECodeChallengeMethodS256)
		}
		if err := nicloudsdk.ValidatePKCECodeChallengeMethod(params.codeChallengeMethod); err != nil {
			httpapi.WriteOAuth2Error(ctx, rw, http.StatusBadRequest, nicloudsdk.OAuth2ErrorCodeInvalidRequest, err.Error())
			return
		}

		// TODO: Ignoring scope for now, but should look into implementing.
		issueAuthCode(rw, r, db, app.ID, apiKey.UserID, params)
	}
}

// issueAuthCode generates an authorization code, writes it to the DB, and redirects to the callback.
func issueAuthCode(rw http.ResponseWriter, r *http.Request, db database.Store, appID uuid.UUID, userID uuid.UUID, params authorizeParams) {
	ctx := r.Context()
	code, err := GenerateSecret()
	if err != nil {
		httpapi.WriteOAuth2Error(ctx, rw, http.StatusInternalServerError, nicloudsdk.OAuth2ErrorCodeServerError, "Failed to generate OAuth2 app authorization code")
		return
	}
	err = db.InTx(func(tx database.Store) error {
		err := tx.DeleteOAuth2ProviderAppCodesByAppAndUserID(ctx, database.DeleteOAuth2ProviderAppCodesByAppAndUserIDParams{
			AppID:  appID,
			UserID: userID,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return xerrors.Errorf("delete oauth2 app codes: %w", err)
		}
		_, err = tx.InsertOAuth2ProviderAppCode(ctx, database.InsertOAuth2ProviderAppCodeParams{
			ID:                  uuid.New(),
			CreatedAt:           dbtime.Now(),
			ExpiresAt:           dbtime.Now().Add(10 * time.Minute),
			SecretPrefix:        []byte(code.Prefix),
			HashedSecret:        code.Hashed,
			AppID:               appID,
			UserID:              userID,
			ResourceUri:         sql.NullString{String: params.resource, Valid: params.resource != ""},
			CodeChallenge:       sql.NullString{String: params.codeChallenge, Valid: params.codeChallenge != ""},
			CodeChallengeMethod: sql.NullString{String: params.codeChallengeMethod, Valid: params.codeChallengeMethod != ""},
			StateHash:           hashOAuth2State(params.state),
			RedirectUri:         sql.NullString{String: params.redirectURL.String(), Valid: params.redirectURIProvided},
		})
		if err != nil {
			return xerrors.Errorf("insert oauth2 authorization code: %w", err)
		}
		return nil
	}, nil)
	if err != nil {
		httpapi.WriteOAuth2Error(ctx, rw, http.StatusInternalServerError, nicloudsdk.OAuth2ErrorCodeServerError, "Failed to generate OAuth2 authorization code")
		return
	}
	newQuery := params.redirectURL.Query()
	newQuery.Add("code", code.Formatted)
	if params.state != "" {
		newQuery.Add("state", params.state)
	}
	params.redirectURL.RawQuery = newQuery.Encode()
	http.Redirect(rw, r, params.redirectURL.String(), http.StatusFound)
}

// hashOAuth2State returns a SHA-256 hash of the OAuth2 state parameter. If
// the state is empty, it returns a null string.
func hashOAuth2State(state string) sql.NullString {
	if state == "" {
		return sql.NullString{}
	}
	hash := sha256.Sum256([]byte(state))
	return sql.NullString{
		String: hex.EncodeToString(hash[:]),
		Valid:  true,
	}
}
