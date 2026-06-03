package httpmw

import (
	"context"
	"net/http"

	"cdr.dev/slog/v3"
	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw/loggermw"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type workspaceParamContextKey struct{}

// WorkspaceParam returns the workspace from the ExtractWorkspaceParam handler.
func WorkspaceParam(r *http.Request) database.Workspace {
	workspace, ok := r.Context().Value(workspaceParamContextKey{}).(database.Workspace)
	if !ok {
		panic("developer error: workspace param middleware not provided")
	}
	return workspace
}

// ExtractWorkspaceParam grabs a workspace from the "workspace" URL parameter.
func ExtractWorkspaceParam(db database.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			workspaceID, parsed := ParseUUIDParam(rw, r, "workspace")
			if !parsed {
				return
			}
			workspace, err := db.GetWorkspaceByID(ctx, workspaceID)
			if httpapi.Is404Error(err) {
				httpapi.ResourceNotFound(rw)
				return
			}
			if err != nil {
				httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
					Message: "Internal error fetching workspace.",
					Detail:  err.Error(),
				})
				return
			}

			ctx = context.WithValue(ctx, workspaceParamContextKey{}, workspace)

			if rlogger := loggermw.RequestLoggerFromContext(ctx); rlogger != nil {
				rlogger.WithFields(slog.F("workspace_name", workspace.Name))
			}

			next.ServeHTTP(rw, r.WithContext(ctx))
		})
	}
}

func WithWorkspaceParam(ctx context.Context, workspace database.Workspace) context.Context {
	return context.WithValue(ctx, workspaceParamContextKey{}, workspace)
}
