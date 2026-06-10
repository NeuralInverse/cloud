package nicloud

import (
	"encoding/binary"
	"encoding/json"
	"net/http"

	"golang.org/x/xerrors"

	"github.com/NeuralInverse/cloud/v2/nicloud/httpapi"
	"github.com/NeuralInverse/cloud/v2/nicloud/httpmw"
)

// giteaUserInfo returns a GitHub-compatible userinfo response for Gitea OAuth2 integration.
// Gitea's "github" provider expects: login, id, email, name, avatar_url.
//
// @Summary Get GitHub-compatible userinfo for Gitea
// @ID gitea-userinfo
// @Security CoderSessionToken
// @Produce json
// @Tags Base
// @Success 200
// @Router /api/v2/gitea/userinfo [get]
func (api *API) giteaUserInfo(rw http.ResponseWriter, r *http.Request) {
	apiKey := httpmw.APIKey(r)
	user, err := api.Database.GetUserByID(r.Context(), apiKey.UserID)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("get user: %w", err))
		return
	}

	// GitHub provider expects id as int — derive a stable int64 from the UUID bytes
	numericID := int64(binary.BigEndian.Uint64(user.ID[:8]))
	if numericID < 0 {
		numericID = -numericID
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(map[string]any{
		"id":         numericID,
		"login":      user.Username,
		"name":       user.Name,
		"email":      user.Email,
		"avatar_url": "",
	})
}
