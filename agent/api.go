package agent

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/NeuralInverse/cloud/v2/agent/agentchat"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw/loggermw"
	"github.com/NeuralInverse/cloud/v2/nicloud/tracing"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk/workspacesdk"
	"github.com/NeuralInverse/cloud/v2/httpmw"
)

func (a *agent) apiHandler() http.Handler {
	r := chi.NewRouter()
	r.Use(
		httpmw.Recover(a.logger),
		tracing.StatusWriterMiddleware,
		loggermw.Logger(a.logger),
		agentchat.Middleware,
	)
	r.Get("/", func(rw http.ResponseWriter, r *http.Request) {
		httpapi.Write(r.Context(), rw, http.StatusOK, nicloudsdk.Response{
			Message: "Hello from the agent!",
		})
	})

	r.Mount("/api/v0", a.filesAPI.Routes())
	r.Mount("/api/v0/git", a.gitAPI.Routes())
	r.Mount("/api/v0/processes", a.processAPI.Routes())
	r.Mount("/api/v0/desktop", a.desktopAPI.Routes())
	r.Mount("/api/v0/mcp", a.mcpAPI.Routes())
	r.Mount("/api/v0/context-config", a.contextConfigAPI.Routes())

	if a.devcontainers {
		r.Mount("/api/v0/containers", a.containerAPI.Routes())
	} else if manifest := a.manifest.Load(); manifest != nil && manifest.ParentID != uuid.Nil {
		r.HandleFunc("/api/v0/containers", func(w http.ResponseWriter, r *http.Request) {
			httpapi.Write(r.Context(), w, http.StatusForbidden, nicloudsdk.Response{
				Message: "Dev Container feature not supported.",
				Detail:  "Dev Container integration inside other Dev Containers is explicitly not supported.",
			})
		})
	} else {
		r.HandleFunc("/api/v0/containers", func(w http.ResponseWriter, r *http.Request) {
			httpapi.Write(r.Context(), w, http.StatusForbidden, nicloudsdk.Response{
				Message: "Dev Container feature not enabled.",
				Detail:  "To enable this feature, set NEURALINVERSE_AGENT_DEVCONTAINERS_ENABLE=true in your template.",
			})
		})
	}

	promHandler := PrometheusMetricsHandler(a.prometheusRegistry, a.logger)

	r.Get("/api/v0/listening-ports", a.listeningPortsHandler.handler)
	r.Get("/api/v0/netcheck", a.HandleNetcheck)
	r.Get("/debug/logs", a.HandleHTTPDebugLogs)
	r.Get("/debug/magicsock", a.HandleHTTPDebugMagicsock)
	r.Get("/debug/magicsock/debug-logging/{state}", a.HandleHTTPMagicsockDebugLoggingState)
	r.Get("/debug/manifest", a.HandleHTTPDebugManifest)
	r.Get("/debug/prometheus", promHandler.ServeHTTP)

	return r
}

type ListeningPortsGetter interface {
	GetListeningPorts() ([]nicloudsdk.WorkspaceAgentListeningPort, error)
}

type listeningPortsHandler struct {
	// In production code, this is set to an osListeningPortsGetter, but it can be overridden for
	// testing.
	getter      ListeningPortsGetter
	ignorePorts map[int]string
}

// handler returns a list of listening ports. This is tested by nicloud's
// TestWorkspaceAgentListeningPorts test.
func (lp *listeningPortsHandler) handler(rw http.ResponseWriter, r *http.Request) {
	ports, err := lp.getter.GetListeningPorts()
	if err != nil {
		httpapi.Write(r.Context(), rw, http.StatusInternalServerError, nicloudsdk.Response{
			Message: "Could not scan for listening ports.",
			Detail:  err.Error(),
		})
		return
	}

	filteredPorts := make([]nicloudsdk.WorkspaceAgentListeningPort, 0, len(ports))
	for _, port := range ports {
		if port.Port < workspacesdk.AgentMinimumListeningPort {
			continue
		}

		// Ignore ports that we've been told to ignore.
		if _, ok := lp.ignorePorts[int(port.Port)]; ok {
			continue
		}
		filteredPorts = append(filteredPorts, port)
	}

	httpapi.Write(r.Context(), rw, http.StatusOK, nicloudsdk.WorkspaceAgentListeningPortsResponse{
		Ports: filteredPorts,
	})
}
