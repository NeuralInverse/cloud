package chatprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/coder/coder/v2/coderd/x/chatd/chatutil"
	"github.com/coder/coder/v2/codersdk"
)

const (
	anthropicThinkingDisplaySummarized = "summarized"
	anthropicThinkingDisplayOmitted    = "omitted"
)

type anthropicThinkingDisplayContextKey struct{}

type anthropicThinkingDisplaySetting struct {
	Display string
}

// ContextWithAnthropicThinkingDisplay adds Anthropic thinking display config to ctx.
func ContextWithAnthropicThinkingDisplay(ctx context.Context, options *codersdk.ChatModelProviderOptions) context.Context {
	display := anthropicThinkingDisplayFromProviderOptions(options)
	if display == nil {
		return ctx
	}
	return context.WithValue(ctx, anthropicThinkingDisplayContextKey{}, anthropicThinkingDisplaySetting{
		Display: *display,
	})
}

func anthropicThinkingDisplayFromProviderOptions(options *codersdk.ChatModelProviderOptions) *string {
	if options == nil || options.Anthropic == nil || options.Anthropic.Thinking == nil {
		return nil
	}
	return anthropicThinkingDisplayFromChat(options.Anthropic.Thinking.Display)
}

func anthropicThinkingDisplayFromChat(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(*value))
	if normalized == "" {
		return nil
	}
	return chatutil.NormalizedEnumValue(
		normalized,
		anthropicThinkingDisplaySummarized,
		anthropicThinkingDisplayOmitted,
	)
}

func withAnthropicThinkingDisplayPatches(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	} else {
		clone := *client
		client = &clone
	}
	client.Transport = &anthropicThinkingDisplayTransport{Base: client.Transport}
	return client
}

type anthropicThinkingDisplayTransport struct {
	Base http.RoundTripper
}

func (t *anthropicThinkingDisplayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base()
	if !shouldPatchAnthropicMessagesRequest(req) {
		return base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	closeErr := req.Body.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}

	patched := patchAnthropicThinkingDisplayBody(body, anthropicThinkingDisplayFromContext(req.Context()))
	patchedReq := req.Clone(req.Context())
	patchedReq.Body = io.NopCloser(bytes.NewReader(patched))
	patchedReq.ContentLength = int64(len(patched))
	patchedReq.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(patched)), nil
	}

	return base.RoundTrip(patchedReq)
}

func (t *anthropicThinkingDisplayTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func shouldPatchAnthropicMessagesRequest(req *http.Request) bool {
	return req != nil &&
		req.Method == http.MethodPost &&
		req.Body != nil &&
		strings.HasSuffix(req.URL.Path, "/v1/messages")
}

func anthropicThinkingDisplayFromContext(ctx context.Context) *anthropicThinkingDisplaySetting {
	setting, ok := ctx.Value(anthropicThinkingDisplayContextKey{}).(anthropicThinkingDisplaySetting)
	if !ok || setting.Display == "" {
		return nil
	}
	return &setting
}

func patchAnthropicThinkingDisplayBody(body []byte, setting *anthropicThinkingDisplaySetting) []byte {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}

	thinkingRaw, ok := payload["thinking"]
	if !ok || len(thinkingRaw) == 0 {
		return body
	}
	var thinking map[string]json.RawMessage
	if err := json.Unmarshal(thinkingRaw, &thinking); err != nil || len(thinking) == 0 {
		return body
	}

	display := ""
	if setting != nil {
		display = setting.Display
	} else if _, ok := thinking["display"]; ok {
		return body
	} else if shouldDefaultAnthropicThinkingDisplay(payload) {
		display = anthropicThinkingDisplaySummarized
	}
	if display == "" {
		return body
	}

	displayRaw, err := json.Marshal(display)
	if err != nil {
		return body
	}
	thinking["display"] = displayRaw

	patchedThinking, err := json.Marshal(thinking)
	if err != nil {
		return body
	}
	payload["thinking"] = patchedThinking

	patched, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return patched
}

func shouldDefaultAnthropicThinkingDisplay(payload map[string]json.RawMessage) bool {
	var model string
	if err := json.Unmarshal(payload["model"], &model); err != nil {
		return false
	}
	return isClaudeOpus48Model(model)
}

func isClaudeOpus48Model(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return model == "claude-opus-4-8" || strings.HasPrefix(model, "claude-opus-4-8-")
}
