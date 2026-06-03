package nicloud

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// RegisterInMemoryAIBridgeProxydHTTPHandler mounts [aibridgeproxyd.Server]'s HTTP handler
// onto [API]'s router, so that requests to aibridgedproxy will be relayed from Neural Inverse Cloud's API server
// to the in-memory aibridgedproxy.
func (api *API) RegisterInMemoryAIBridgeProxydHTTPHandler(srv http.Handler) {
	if srv == nil {
		panic("aibridgeproxyd cannot be nil")
	}

	api.aibridgeproxydHandler = srv
}

// aibridgeproxyHandler handles AI Bridge Proxy endpoints.
func aibridgeproxyHandler(api *API, middlewares ...func(http.Handler) http.Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(api.RequireFeatureMW(nicloudsdk.FeatureAIBridge))
		r.Use(middlewares...)

		r.HandleFunc("/*", func(rw http.ResponseWriter, r *http.Request) {
			// Check if the proxy is enabled.
			if !api.DeploymentValues.AI.BridgeProxyConfig.Enabled.Value() {
				httpapi.Write(r.Context(), rw, http.StatusNotFound, nicloudsdk.Response{
					Message: "AI Bridge Proxy is not enabled.",
				})
				return
			}

			// Check if the handler is registered.
			if api.aibridgeproxydHandler == nil {
				httpapi.Write(r.Context(), rw, http.StatusNotFound, nicloudsdk.Response{
					Message: "AI Bridge Proxy handler not mounted.",
				})
				return
			}

			// Strip the prefix and relay to the aibridgeproxyd handler.
			http.StripPrefix("/api/v2/aibridge/proxy", api.aibridgeproxydHandler).ServeHTTP(rw, r)
		})
	}
}
