package httperror

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type Responder interface {
	Response() (int, nicloudsdk.Response)
}

func IsResponder(err error) (Responder, bool) {
	var responseErr Responder
	if errors.As(err, &responseErr) {
		return responseErr, true
	}
	return nil, false
}

func NewResponseError(status int, resp nicloudsdk.Response) error {
	return &responseError{
		status:   status,
		response: resp,
	}
}

func WriteResponseError(ctx context.Context, rw http.ResponseWriter, err error) {
	if responseErr, ok := IsResponder(err); ok {
		code, resp := responseErr.Response()

		httpapi.Write(ctx, rw, code, resp)
		return
	}

	httpapi.Write(ctx, rw, http.StatusInternalServerError, nicloudsdk.Response{
		Message: "Internal server error",
		Detail:  err.Error(),
	})
}

type responseError struct {
	status   int
	response nicloudsdk.Response
}

var (
	_ error     = (*responseError)(nil)
	_ Responder = (*responseError)(nil)
)

func (e *responseError) Error() string {
	return fmt.Sprintf("%s: %s", e.response.Message, e.response.Detail)
}

func (e *responseError) Status() int {
	return e.status
}

func (e *responseError) Response() (int, nicloudsdk.Response) {
	return e.status, e.response
}

var ErrResourceNotFound = NewResponseError(http.StatusNotFound, httpapi.ResourceNotFoundResponse)
