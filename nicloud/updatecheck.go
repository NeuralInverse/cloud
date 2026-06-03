package nicloud

import (
	"database/sql"
	"net/http"
	"strings"

	"golang.org/x/mod/semver"
	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/buildinfo"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloudsdk"
)

// @Summary Update check
// @ID update-check
// @Produce json
// @Tags General
// @Success 200 {object} nicloudsdk.UpdateCheckResponse
// @Router /api/v2/updatecheck [get]
func (api *API) updateCheck(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	currentVersion := nicloudsdk.UpdateCheckResponse{
		Current: true,
		Version: buildinfo.Version(),
		URL:     buildinfo.ExternalURL(),
	}

	if api.updateChecker == nil {
		// If update checking is disabled, echo the current
		// version.
		httpapi.Write(ctx, rw, http.StatusOK, currentVersion)
		return
	}

	uc, err := api.updateChecker.Latest(ctx)
	if err != nil {
		if xerrors.Is(err, sql.ErrNoRows) {
			// Update checking is enabled, but has never
			// succeeded, reproduce behavior as if disabled.
			httpapi.Write(ctx, rw, http.StatusOK, currentVersion)
			return
		}

		httpapi.InternalServerError(rw, err)
		return
	}

	// Since our dev version (v0.12.9-devel+f7246386) is not semver compatible,
	// ignore everything after "-"."
	versionWithoutDevel := strings.SplitN(buildinfo.Version(), "-", 2)[0]

	httpapi.Write(ctx, rw, http.StatusOK, nicloudsdk.UpdateCheckResponse{
		Current: semver.Compare(versionWithoutDevel, uc.Version) >= 0,
		Version: uc.Version,
		URL:     uc.URL,
	})
}
