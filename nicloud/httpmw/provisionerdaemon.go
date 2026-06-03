package httpmw

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/NeuralInverse/cloud/v2/nicloud/database"
	"github.com/NeuralInverse/cloud/v2/nicloud/database/dbauthz"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/provisionerkey"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

type provisionerDaemonContextKey struct{}

func ProvisionerDaemonAuthenticated(r *http.Request) bool {
	proxy, ok := r.Context().Value(provisionerDaemonContextKey{}).(bool)
	return ok && proxy
}

type ExtractProvisionerAuthConfig struct {
	DB       database.Store
	Optional bool
	PSK      string
}

// ExtractProvisionerDaemonAuthenticated authenticates a request as a provisioner daemon.
// If the request is not authenticated, the next handler is called unless Optional is true.
// This function currently is tested inside the enterprise package.
func ExtractProvisionerDaemonAuthenticated(opts ExtractProvisionerAuthConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			handleOptional := func(code int, response nicloudsdk.Response) {
				if opts.Optional {
					next.ServeHTTP(w, r)
					return
				}
				httpapi.Write(ctx, w, code, response)
			}

			psk := r.Header.Get(nicloudsdk.ProvisionerDaemonPSK)
			key := r.Header.Get(nicloudsdk.ProvisionerDaemonKey)
			if key == "" {
				if opts.PSK == "" {
					handleOptional(http.StatusUnauthorized, nicloudsdk.Response{
						Message: "provisioner daemon key required",
					})
					return
				}

				fallbackToPSK(ctx, opts.PSK, next, w, r, handleOptional)
				return
			}
			if psk != "" {
				handleOptional(http.StatusBadRequest, nicloudsdk.Response{
					Message: "provisioner daemon key and psk provided, but only one is allowed",
				})
				return
			}

			err := provisionerkey.Validate(key)
			if err != nil {
				handleOptional(http.StatusBadRequest, nicloudsdk.Response{
					Message: "provisioner daemon key invalid",
					Detail:  err.Error(),
				})
				return
			}
			hashedKey := provisionerkey.HashSecret(key)
			// nolint:gocritic // System must check if the provisioner key is valid.
			pk, err := opts.DB.GetProvisionerKeyByHashedSecret(dbauthz.AsSystemRestricted(ctx), hashedKey)
			if err != nil {
				if httpapi.Is404Error(err) {
					handleOptional(http.StatusUnauthorized, nicloudsdk.Response{
						Message: "provisioner daemon key invalid",
					})
					return
				}

				handleOptional(http.StatusInternalServerError, nicloudsdk.Response{
					Message: "get provisioner daemon key",
					Detail:  err.Error(),
				})
				return
			}

			if provisionerkey.Compare(pk.HashedSecret, hashedKey) {
				handleOptional(http.StatusUnauthorized, nicloudsdk.Response{
					Message: "provisioner daemon key invalid",
				})
				return
			}

			// The provisioner key does not indicate a specific provisioner daemon. So just
			// store a boolean so the caller can check if the request is from an
			// authenticated provisioner daemon.
			ctx = context.WithValue(ctx, provisionerDaemonContextKey{}, true)
			// store key used to authenticate the request
			ctx = context.WithValue(ctx, provisionerKeyAuthContextKey{}, pk)
			// nolint:gocritic // Authenticating as a provisioner daemon.
			ctx = dbauthz.AsProvisionerd(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type provisionerKeyAuthContextKey struct{}

func ProvisionerKeyAuthOptional(r *http.Request) (database.ProvisionerKey, bool) {
	user, ok := r.Context().Value(provisionerKeyAuthContextKey{}).(database.ProvisionerKey)
	return user, ok
}

func fallbackToPSK(ctx context.Context, psk string, next http.Handler, w http.ResponseWriter, r *http.Request, handleOptional func(code int, response nicloudsdk.Response)) {
	token := r.Header.Get(nicloudsdk.ProvisionerDaemonPSK)
	if subtle.ConstantTimeCompare([]byte(token), []byte(psk)) != 1 {
		handleOptional(http.StatusUnauthorized, nicloudsdk.Response{
			Message: "provisioner daemon psk invalid",
		})
		return
	}

	// The PSK does not indicate a specific provisioner daemon. So just
	// store a boolean so the caller can check if the request is from an
	// authenticated provisioner daemon.
	ctx = context.WithValue(ctx, provisionerDaemonContextKey{}, true)
	// nolint:gocritic // Authenticating as a provisioner daemon.
	ctx = dbauthz.AsProvisionerd(ctx)
	next.ServeHTTP(w, r.WithContext(ctx))
}
