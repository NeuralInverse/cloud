package httperror

import (
	"context"
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

func WriteWorkspaceBuildError(ctx context.Context, rw http.ResponseWriter, err error) {
	if responseErr, ok := IsResponder(err); ok {
		code, resp := responseErr.Response()

		httpapi.Write(ctx, rw, code, resp)
		return
	}

	httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
		Message: "Internal error creating workspace build.",
		Detail:  err.Error(),
	})
}
