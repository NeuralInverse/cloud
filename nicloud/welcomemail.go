package nicloud

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"os"
	"strings"

	"cdr.dev/slog/v3"
)

func sendWelcomeEmail(_ context.Context, logger slog.Logger, email, name string) {
	ctx := context.Background()
	apiKey := os.Getenv("NEURALINVERSE_RESEND_API_KEY")
	if apiKey == "" {
		logger.Warn(ctx, "skipping welcome email, NEURALINVERSE_RESEND_API_KEY not set")
		return
	}

	firstName := name
	if idx := strings.Index(name, " "); idx > 0 {
		firstName = name[:idx]
	}
	if firstName == "" {
		firstName = strings.Split(email, "@")[0]
	}

	htmlBody := buildWelcomeHTML(firstName)

	subjects := []string{
		"Free AI models. No rate limits. Forever.",
		"Your Neural Inverse Cloud account is ready",
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(subjects))))
	subject := subjects[n.Int64()]

	payload := map[string]any{
		"from":    "Neural Inverse <cloud@noreply.neuralinverse.io>",
		"to":     []string{email},
		"bcc":    []string{"vakeesh@hello.neuralinverse.io", "vakeesh@neuralinverse.com"},
		"subject": subject,
		"html":   htmlBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Warn(ctx, "failed to marshal welcome email payload", slog.Error(err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		logger.Warn(ctx, "failed to create welcome email request", slog.Error(err))
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Warn(ctx, "failed to send welcome email", slog.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Warn(ctx, "welcome email API returned error", slog.F("status", resp.StatusCode))
		return
	}

	logger.Info(ctx, "welcome email sent", slog.F("to", email))
}

func buildWelcomeHTML(firstName string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Neural Inverse Cloud - You're in</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #ffffff;">
    <table role="presentation" style="width: 100%; border-collapse: collapse;">
        <tr>
            <td style="padding: 40px 20px;">
                <table role="presentation" style="max-width: 560px; margin: 0 auto;">

                    <tr>
                        <td style="padding: 0 20px 40px;">
                            <img src="https://cdn.neuralinverse.io/logo.png" alt="Neural Inverse" style="width: 40px; height: auto; margin-bottom: 32px;">

                            <h1 style="margin: 0 0 24px; font-size: 28px; font-weight: 600; color: #000000; line-height: 1.2;">
                                You're in, <span style="background-color: #fff59d; padding: 2px 0;">` + firstName + `</span>
                            </h1>

                            <p style="margin: 0 0 24px; font-size: 16px; line-height: 1.6; color: #333333;">Neural Inverse Cloud is the open-source cloud IDE for regulated software. AGPL-licensed. No enterprise upsell. Pick your region - we host it so you don't have to.</p>

                            <div style="margin-bottom: 32px; padding: 20px; background-color: #fff59d; border-radius: 4px;">
                                <p style="margin: 0 0 8px; font-size: 18px; font-weight: 600; color: #000000;">$1.22 credit waiting</p>
                                <p style="margin: 0 0 16px; font-size: 15px; line-height: 1.5; color: #333333;">
                                    No card needed. Activate billing &rarr; unlock $10 more.<br>
                                    Then $0.24/core-hr + $0.06/GB-hr. Pay only when running.
                                </p>
                                <a href="https://cloud.neuralinverse.com/login?redirect=/api/v2/billing/redirect" style="display: inline-block; padding: 10px 20px; background-color: #000000; color: #ffffff; text-decoration: none; border-radius: 4px; font-size: 14px; font-weight: 500;">Activate Billing ($10 Free)</a>
                            </div>

                            <table role="presentation" style="width: 100%; margin-bottom: 32px;">
                                <tr>
                                    <td>
                                        <a href="https://neuralinverse.com/cloud" style="display: inline-block; padding: 14px 28px; background-color: #000000; color: #ffffff; text-decoration: none; border-radius: 4px; font-size: 16px; font-weight: 500;">Choose Region & Launch</a>
                                    </td>
                                </tr>
                            </table>

                            <p style="margin: 0 0 16px; font-size: 15px; line-height: 1.7; color: #333333;">
                                <strong>What's included:</strong><br>
                                AI models are free. Forever. No rate limits.<br>
                                Multi-region deployment (US, Europe, Singapore, Japan)<br>
                                Git hosting with LFS (multi-region)<br>
                                Bring your own LLM
                            </p>

                            <div style="margin: 32px 0 0; padding-top: 24px; border-top: 1px solid #e0e0e0;">
                                <p style="margin: 0 0 12px; font-size: 14px; line-height: 1.6; color: #666666;">
                                    Trusted by 1,000+ developers. If you hit a wall, email me.
                                </p>
                                <p style="margin: 0; font-size: 14px; color: #666666;">
                                    - Vakeesh<br>
                                    <span style="font-size: 12px; color: #999999;">Co-founder, Neural Inverse</span><br>
                                    <a href="mailto:vakeesh@neuralinverse.com" style="color: #000000; text-decoration: none; font-weight: 500;">vakeesh@neuralinverse.com</a>
                                </p>
                            </div>

                        </td>
                    </tr>

                    <tr>
                        <td style="padding: 24px 20px 40px; text-align: center; font-size: 12px; color: #999999;">
                            <a href="https://github.com/NeuralInverse/neuralinverse" style="color: #666666; text-decoration: none; margin: 0 12px; font-weight: 500;">GitHub</a>
                            <span style="color: #cccccc;">|</span>
                            <a href="https://neuralinverse.com/docs/cloud" style="color: #666666; text-decoration: none; margin: 0 12px; font-weight: 500;">Docs</a>
                            <span style="color: #cccccc;">|</span>
                            <a href="https://discord.gg/tsFRzk9h" style="color: #666666; text-decoration: none; margin: 0 12px; font-weight: 500;">Discord</a>
                            <p style="margin: 12px 0 0;">&copy; 2026 Neural Inverse. All rights reserved - <a href="https://cloud.neuralinverse.com" style="color: #666666; text-decoration: none;">https://cloud.neuralinverse.com</a></p>
                            <a href="https://cloud.neuralinverse.com/settings/notifications" style="color: #999999; text-decoration: underline; display: block; margin-top: 8px;">Stop receiving emails like this</a>
                        </td>
                    </tr>

                </table>
            </td>
        </tr>
    </table>
</body>
</html>`
}
