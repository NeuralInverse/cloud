package nicloud

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

// baseWorkspaceInitResponse is returned by the workspace-init endpoint.
type baseWorkspaceInitResponse struct {
	RepoURL    string `json:"repo_url"`
	GitToken   string `json:"git_token"`
	RepoExists bool   `json:"repo_exists"`
}

// baseWorkspaceInit is called from the workspace startup_script to:
//  1. Ensure the user has a Gitea account (auto-created via OAuth2 on first Base login).
//  2. Create a private repo named after the workspace if it doesn't exist.
//  3. Return a short-lived Gitea user token scoped to that repo.
//
// @Summary Initialize workspace repo on Base
// @ID base-workspace-init
// @Security CoderSessionToken
// @Produce json
// @Tags Base
// @Param workspace_name query string true "Workspace name"
// @Success 200 {object} baseWorkspaceInitResponse
// @Router /api/v2/base/workspace-init [get]
func (api *API) baseWorkspaceInit(rw http.ResponseWriter, r *http.Request) {
	baseURL := api.DeploymentValues.BaseURL.String()
	adminToken := api.DeploymentValues.BaseAdminToken.String()
	if baseURL == "" || adminToken == "" {
		http.NotFound(rw, r)
		return
	}

	apiKey := httpmw.APIKey(r)
	user, err := api.Database.GetUserByID(r.Context(), apiKey.UserID)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("get user: %w", err))
		return
	}

	workspaceName := r.URL.Query().Get("workspace_name")
	if workspaceName == "" {
		http.Error(rw, "workspace_name required", http.StatusBadRequest)
		return
	}

	gc := &giteaAdminClient{baseURL: strings.TrimRight(baseURL, "/"), token: adminToken}

	// Ensure Gitea user exists (may not have logged into Base yet)
	if err := gc.ensureUser(user.Username, user.Email, user.Name); err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("ensure gitea user: %w", err))
		return
	}

	// Create private repo for this workspace if needed
	repoName := workspaceName
	repoExists, err := gc.ensureRepo(user.Username, repoName)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("ensure gitea repo: %w", err))
		return
	}

	// Issue a user token scoped to this session
	tokenName := fmt.Sprintf("workspace-%s-%d", workspaceName, time.Now().Unix())
	gitToken, err := gc.createUserToken(user.Username, tokenName)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("create gitea token: %w", err))
		return
	}

	repoURL := fmt.Sprintf("%s/%s/%s.git", baseURL, user.Username, repoName)

	httpapi.Write(r.Context(), rw, http.StatusOK, baseWorkspaceInitResponse{
		RepoURL:    repoURL,
		GitToken:   gitToken,
		RepoExists: repoExists,
	})
}

// giteaAdminClient calls the Gitea admin API on behalf of Cloud.
type giteaAdminClient struct {
	baseURL string
	token   string
}

func (g *giteaAdminClient) do(method, path string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, g.baseURL+"/api/v1"+path, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "token "+g.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

func (g *giteaAdminClient) ensureUser(username, email, fullName string) error {
	_, status, err := g.do("GET", "/users/"+username, nil)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil // already exists
	}
	// Create via admin API — no password needed (OAuth2-only login)
	_, status, err = g.do("POST", "/admin/users", map[string]any{
		"username":             username,
		"email":                email,
		"full_name":            fullName,
		"login_name":           username,
		"source_id":            0,
		"must_change_password": false,
		"send_notify":          false,
		"password":             randomHex(16),
	})
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return xerrors.Errorf("create gitea user status %d", status)
	}
	return nil
}

func (g *giteaAdminClient) ensureRepo(owner, repo string) (exists bool, err error) {
	_, status, err := g.do("GET", fmt.Sprintf("/repos/%s/%s", owner, repo), nil)
	if err != nil {
		return false, err
	}
	if status == http.StatusOK {
		return true, nil
	}
	// Create private repo
	_, status, err = g.do("POST", fmt.Sprintf("/admin/users/%s/repos", owner), map[string]any{
		"name":          repo,
		"private":       true,
		"auto_init":     true,
		"default_branch": "main",
	})
	if err != nil {
		return false, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return false, xerrors.Errorf("create gitea repo status %d", status)
	}
	return false, nil
}

func (g *giteaAdminClient) createUserToken(username, tokenName string) (string, error) {
	// Delete old token with same name to avoid conflicts
	_, _, _ = g.do("DELETE", fmt.Sprintf("/users/%s/tokens/%s", username, tokenName), nil)

	b, status, err := g.do("POST", fmt.Sprintf("/users/%s/tokens", username), map[string]any{
		"name": tokenName,
	})
	if err != nil {
		return "", err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return "", xerrors.Errorf("create gitea token status %d", status)
	}
	var resp struct {
		SHA1 string `json:"sha1"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", err
	}
	return resp.SHA1, nil
}

func randomHex(n int) string {
	const letters = "0123456789abcdef"
	b := make([]byte, n*2)
	for i := range b {
		b[i] = letters[i%16]
	}
	return string(b)
}
