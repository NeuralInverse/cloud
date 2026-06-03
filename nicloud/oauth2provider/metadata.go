package oauth2provider

import (
	"net/http"
	"net/url"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/rbac"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// GetAuthorizationServerMetadata returns an http.HandlerFunc that handles GET /.well-known/oauth-authorization-server
func GetAuthorizationServerMetadata(accessURL *url.URL) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		metadata := nicloudsdk.OAuth2AuthorizationServerMetadata{
			Issuer:                            accessURL.String(),
			AuthorizationEndpoint:             accessURL.JoinPath("/oauth2/authorize").String(),
			TokenEndpoint:                     accessURL.JoinPath("/oauth2/tokens").String(),
			RegistrationEndpoint:              accessURL.JoinPath("/oauth2/register").String(), // RFC 7591
			RevocationEndpoint:                accessURL.JoinPath("/oauth2/revoke").String(),   // RFC 7009
			ResponseTypesSupported:            []nicloudsdk.OAuth2ProviderResponseType{nicloudsdk.OAuth2ProviderResponseTypeCode},
			GrantTypesSupported:               []nicloudsdk.OAuth2ProviderGrantType{nicloudsdk.OAuth2ProviderGrantTypeAuthorizationCode, nicloudsdk.OAuth2ProviderGrantTypeRefreshToken},
			CodeChallengeMethodsSupported:     []nicloudsdk.OAuth2PKCECodeChallengeMethod{nicloudsdk.OAuth2PKCECodeChallengeMethodS256},
			ScopesSupported:                   rbac.ExternalScopeNames(),
			TokenEndpointAuthMethodsSupported: []nicloudsdk.OAuth2TokenEndpointAuthMethod{nicloudsdk.OAuth2TokenEndpointAuthMethodClientSecretBasic, nicloudsdk.OAuth2TokenEndpointAuthMethodClientSecretPost},
		}
		httpapi.Write(ctx, rw, http.StatusOK, metadata)
	}
}

// GetProtectedResourceMetadata returns an http.HandlerFunc that handles GET /.well-known/oauth-protected-resource
func GetProtectedResourceMetadata(accessURL *url.URL) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		metadata := nicloudsdk.OAuth2ProtectedResourceMetadata{
			Resource:             accessURL.String(),
			AuthorizationServers: []string{accessURL.String()},
			ScopesSupported:      rbac.ExternalScopeNames(),
			// RFC 6750 Bearer Token methods supported as fallback methods in api key middleware
			BearerMethodsSupported: []string{"header", "query"},
		}
		httpapi.Write(ctx, rw, http.StatusOK, metadata)
	}
}
