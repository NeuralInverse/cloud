package nicloud

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	RepoURL       string `json:"repo_url"`
	GitToken      string `json:"git_token"`
	RepoExists    bool   `json:"repo_exists"`
	RegistryToken string `json:"registry_token"`
	RegistryURL   string `json:"registry_url"`
	RunnerToken   string `json:"runner_token"`
	ActionsURL    string `json:"actions_url"`
}

// baseWorkspaceInit is called from the workspace startup_script to:
//  1. Ensure the user has a Gitea account.
//  2. Create a private repo named after the workspace if it doesn't exist.
//  3. Return a Gitea user token (git + registry scopes).
//  4. Return a runner registration token for act_runner.
//  5. Register a webhook on the repo to call back into Cloud on push.
//
// @Summary Initialize workspace repo on Base
// @ID base-workspace-init
// @Security CoderSessionToken
// @Produce json
// @Tags Base
// @Success 200 {object} baseWorkspaceInitResponse
// @Router /api/v2/base/workspace-init [get]
func (api *API) baseWorkspaceInit(rw http.ResponseWriter, r *http.Request) {
	baseURL := api.DeploymentValues.BaseURL.String()
	adminToken := api.DeploymentValues.BaseAdminToken.String()
	if baseURL == "" || adminToken == "" {
		http.NotFound(rw, r)
		return
	}

	agent := httpmw.WorkspaceAgent(r)
	workspace, err := api.Database.GetWorkspaceByAgentID(r.Context(), agent.ID)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("get workspace: %w", err))
		return
	}
	user, err := api.Database.GetUserByID(r.Context(), workspace.OwnerID)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("get user: %w", err))
		return
	}

	workspaceName := workspace.Name
	if workspaceName == "" {
		http.Error(rw, "workspace_name required", http.StatusBadRequest)
		return
	}

	gc := &giteaAdminClient{
		baseURL:   strings.TrimRight(baseURL, "/"),
		token:     adminToken,
		adminUser: api.DeploymentValues.BaseAdminUser.String(),
		adminPass: api.DeploymentValues.BaseAdminPass.String(),
	}

	if err := gc.ensureUser(user.Username, user.Email, user.Name); err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("ensure gitea user: %w", err))
		return
	}

	repoExists, err := gc.ensureRepo(user.Username, workspaceName)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("ensure gitea repo: %w", err))
		return
	}

	// Register webhook on new repos so Cloud gets push notifications.
	webhookSecret := api.DeploymentValues.BaseWebhookSecret.String()
	cloudURL := api.DeploymentValues.AccessURL.String()
	if !repoExists && webhookSecret != "" && cloudURL != "" {
		_ = gc.ensureWebhook(user.Username, workspaceName,
			strings.TrimRight(cloudURL, "/")+"/api/v2/base/webhook",
			webhookSecret,
		)
	}

	tokenName := fmt.Sprintf("workspace-%s-%d", workspaceName, time.Now().Unix())
	gitToken, err := gc.createUserToken(user.Username, tokenName,
		[]string{"write:repository", "write:user", "write:package", "read:package"},
	)
	if err != nil {
		httpapi.InternalServerError(rw, xerrors.Errorf("create gitea token: %w", err))
		return
	}

	// Runner registration token — best-effort, empty string if actions disabled.
	runnerToken, _ := gc.createRunnerToken(user.Username, workspaceName)

	repoURL := fmt.Sprintf("%s/%s/%s.git", strings.TrimRight(baseURL, "/"), user.Username, workspaceName)
	registryURL := api.DeploymentValues.BaseRegistryURL.String()
	actionsURL := api.DeploymentValues.BaseActionsURL.String()

	httpapi.Write(r.Context(), rw, http.StatusOK, baseWorkspaceInitResponse{
		RepoURL:       repoURL,
		GitToken:      gitToken,
		RepoExists:    repoExists,
		RegistryToken: gitToken, // same token — Gitea accepts it for package auth
		RegistryURL:   registryURL,
		RunnerToken:   runnerToken,
		ActionsURL:    actionsURL,
	})
}

// baseWebhookPayload is the subset of Gitea's push event we care about.
type baseWebhookPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Pusher struct {
		Login string `json:"login"`
	} `json:"pusher"`
}

// baseWebhook receives push events from Gitea and can trigger downstream actions.
//
// @Summary Receive Gitea push webhook
// @ID base-webhook
// @Tags Base
// @Success 204
// @Router /api/v2/base/webhook [post]
func (api *API) baseWebhook(rw http.ResponseWriter, r *http.Request) {
	secret := api.DeploymentValues.BaseWebhookSecret.String()
	if secret == "" {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(rw, "read body", http.StatusBadRequest)
		return
	}

	sig := strings.TrimPrefix(r.Header.Get("X-Gitea-Signature"), "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		http.Error(rw, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload baseWebhookPayload
	_ = json.Unmarshal(body, &payload)

	// Log the push — extend here to trigger workspace rebuilds, notifications, etc.
	api.Logger.Info(r.Context(), "base push webhook",
		"repo", payload.Repository.FullName,
		"ref", payload.Ref,
		"pusher", payload.Pusher.Login,
	)

	rw.WriteHeader(http.StatusNoContent)
}

// giteaAdminClient calls the Gitea admin API on behalf of Cloud.
type giteaAdminClient struct {
	baseURL   string
	token     string
	adminUser string
	adminPass string
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
		return nil
	}
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
	_, status, err = g.do("POST", fmt.Sprintf("/admin/users/%s/repos", owner), map[string]any{
		"name":           repo,
		"private":        true,
		"auto_init":      true,
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

func (g *giteaAdminClient) ensureWebhook(owner, repo, targetURL, secret string) error {
	// Check if our webhook already exists to avoid duplicates.
	b, status, err := g.do("GET", fmt.Sprintf("/repos/%s/%s/hooks", owner, repo), nil)
	if err != nil || status != http.StatusOK {
		return xerrors.Errorf("list hooks status %d: %w", status, err)
	}
	var hooks []struct {
		Config struct {
			URL string `json:"url"`
		} `json:"config"`
	}
	_ = json.Unmarshal(b, &hooks)
	for _, h := range hooks {
		if h.Config.URL == targetURL {
			return nil // already registered
		}
	}

	_, status, err = g.do("POST", fmt.Sprintf("/repos/%s/%s/hooks", owner, repo), map[string]any{
		"type":   "gitea",
		"active": true,
		"events": []string{"push"},
		"config": map[string]string{
			"url":          targetURL,
			"content_type": "json",
			"secret":       secret,
		},
	})
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return xerrors.Errorf("create hook status %d", status)
	}
	return nil
}

func (g *giteaAdminClient) createRunnerToken(owner, repo string) (string, error) {
	b, status, err := g.do("GET", fmt.Sprintf("/repos/%s/%s/actions/runners/registration-token", owner, repo), nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", xerrors.Errorf("runner token status %d", status)
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return "", err
	}
	return resp.Token, nil
}

func (g *giteaAdminClient) createUserToken(username, tokenName string, scopes []string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"name":   tokenName,
		"scopes": scopes,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", g.baseURL+"/api/v1/users/"+username+"/tokens", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(g.adminUser, g.adminPass)
	req.Header.Set("Sudo", username)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", xerrors.Errorf("create gitea token status %d", resp.StatusCode)
	}
	var tresp struct {
		SHA1 string `json:"sha1"`
	}
	if err := json.Unmarshal(b, &tresp); err != nil {
		return "", err
	}
	return tresp.SHA1, nil
}

func randomHex(n int) string {
	const letters = "0123456789abcdef"
	b := make([]byte, n*2)
	for i := range b {
		b[i] = letters[i%16]
	}
	return string(b)
}
