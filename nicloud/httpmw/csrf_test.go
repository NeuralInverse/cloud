package httpmw_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justinas/nosurf"
	"github.com/stretchr/testify/require"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func TestCSRFExemptList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name   string
		URL    string
		Exempt bool
	}{
		{
			Name:   "Root",
			URL:    "https://example.com",
			Exempt: true,
		},
		{
			Name:   "WorkspacePage",
			URL:    "https://cloud.neuralinverse.com/workspaces",
			Exempt: true,
		},
		{
			Name:   "SubApp",
			URL:    "https://app--dev--coder--user--apps.cloud.neuralinverse.com/",
			Exempt: true,
		},
		{
			Name:   "PathApp",
			URL:    "https://cloud.neuralinverse.com/@USER/test.instance/apps/app",
			Exempt: true,
		},
		{
			Name:   "API",
			URL:    "https://cloud.neuralinverse.com/api/v2",
			Exempt: false,
		},
		{
			Name:   "APIMe",
			URL:    "https://cloud.neuralinverse.com/api/v2/me",
			Exempt: false,
		},
		{
			Name:   "OAuth2Authorize",
			URL:    "https://cloud.neuralinverse.com/oauth2/authorize",
			Exempt: false,
		},
		{
			Name:   "OAuth2AuthorizeQuery",
			URL:    "https://cloud.neuralinverse.com/oauth2/authorize?client_id=test",
			Exempt: false,
		},
		{
			Name:   "OAuth2Tokens",
			URL:    "https://cloud.neuralinverse.com/oauth2/tokens",
			Exempt: true,
		},
		{
			Name:   "OAuth2Register",
			URL:    "https://cloud.neuralinverse.com/oauth2/register",
			Exempt: true,
		},
	}

	mw := httpmw.CSRF(nicloudsdk.HTTPCookieConfig{})
	csrfmw := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).(*nosurf.CSRFHandler)

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			r, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.URL, nil)
			require.NoError(t, err)

			r.AddCookie(&http.Cookie{Name: nicloudsdk.SessionTokenCookie, Value: "test"})
			exempt := csrfmw.IsExempt(r)
			require.Equal(t, c.Exempt, exempt)
		})
	}
}

// TestCSRFError verifies the error message returned to a user when CSRF
// checks fail.
//
//nolint:bodyclose // Using httptest.Recorders
func TestCSRFError(t *testing.T) {
	t.Parallel()

	// Hard coded matching CSRF values
	const csrfCookieValue = "JXm9hOUdZctWt0ZZGAy9xiS/gxMKYOThdxjjMnMUyn4="
	const csrfHeaderValue = "KNKvagCBEHZK7ihe2t7fj6VeJ0UyTDco1yVUJE8N06oNqxLu5Zx1vRxZbgfC0mJJgeGkVjgs08mgPbcWPBkZ1A=="
	// Use a url with "/api" as the root, other routes bypass CSRF.
	const urlPath = "https://cloud.neuralinverse.com/api/v2/hello"

	var handler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	handler = httpmw.CSRF(nicloudsdk.HTTPCookieConfig{})(handler)

	// Not testing the error case, just providing the example of things working
	// to base the failure tests off of.
	t.Run("ValidCSRF", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlPath, nil)
		require.NoError(t, err)

		req.AddCookie(&http.Cookie{Name: nicloudsdk.SessionTokenCookie, Value: "session_token_value"})
		req.AddCookie(&http.Cookie{Name: nosurf.CookieName, Value: csrfCookieValue})
		req.Header.Add(nosurf.HeaderName, csrfHeaderValue)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		resp := rec.Result()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// The classic CSRF failure returns the generic error.
	t.Run("MissingCSRFHeader", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlPath, nil)
		require.NoError(t, err)

		req.AddCookie(&http.Cookie{Name: nicloudsdk.SessionTokenCookie, Value: "session_token_value"})
		req.AddCookie(&http.Cookie{Name: nosurf.CookieName, Value: csrfCookieValue})

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		resp := rec.Result()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, rec.Body.String(), "Something is wrong with your CSRF token.")
	})

	// Include the CSRF cookie, but not the CSRF header value.
	// Including the 'nicloudsdk.SessionTokenHeader' will bypass CSRF only if
	// it matches the cookie. If it does not, we expect a more helpful error.
	t.Run("MismatchedHeaderAndCookie", func(t *testing.T) {
		t.Parallel()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlPath, nil)
		require.NoError(t, err)

		req.AddCookie(&http.Cookie{Name: nicloudsdk.SessionTokenCookie, Value: "session_token_value"})
		req.AddCookie(&http.Cookie{Name: nosurf.CookieName, Value: csrfCookieValue})
		req.Header.Add(nicloudsdk.SessionTokenHeader, "mismatched_value")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		resp := rec.Result()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.Contains(t, rec.Body.String(), "CSRF error encountered. Authentication via")
	})
}
