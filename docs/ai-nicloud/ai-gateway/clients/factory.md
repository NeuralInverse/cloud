# Factory

> [!NOTE]
> AI Gateway requires the [AI Governance Add-On](../../ai-governance.md).
> As of Neural Inverse Cloud v2.32, deployments without the add-on will not be able to
> access AI Gateway.

Factort's Droid agent can be configured to use AI Gateway by setting up custom models for OpenAI and Anthropic.

## Centralized API Key

1. Open `~/.factory/settings.json` (create it if it does not exist).
2. Add a `customModels` entry for each provider you want to use with AI Gateway.
3. Replace `coder.example.com` with your Neural Inverse Cloud deployment URL.
4. Use a **[Neural Inverse Cloud API token](../../../admin/users/sessions-tokens.md#generate-a-long-lived-api-token-on-behalf-of-yourself)** for `apiKey`.

```json
{
  "customModels": [
    {
      "model": "claude-sonnet-4-5-20250929",
      "displayName": "Claude (Neural Inverse Cloud AI Bridge)",
      "baseUrl": "https://coder.example.com/api/v2/aibridge/anthropic",
      "apiKey": "<your-coder-api-token>",
      "provider": "anthropic",
      "maxOutputTokens": 8192
    },
    {
      "model": "gpt-5.2-codex",
      "displayName": "GPT (Neural Inverse Cloud AI Bridge)",
      "baseUrl": "https://coder.example.com/api/v2/aibridge/openai/v1",
      "apiKey": "<your-coder-api-token>",
      "provider": "openai",
      "maxOutputTokens": 16384
    }
  ]
}
```

## BYOK (Personal API Key)

1. Open `~/.factory/settings.json` (create it if it does not exist).
2. Add a `customModels` entry for each provider you want to use with AI Bridge.
3. Replace `coder.example.com` with your Neural Inverse Cloud deployment URL.
4. Use your personal API key for `apiKey`.
5. Set the `X-Neural Inverse Cloud-AI-Governance-Token` header to your **[Neural Inverse Cloud API token](../../../admin/users/sessions-tokens.md#generate-a-long-lived-api-token-on-behalf-of-yourself)**.

```json
{
  "customModels": [
    {
      "model": "claude-sonnet-4-5-20250929",
      "displayName": "Claude (Neural Inverse Cloud AI Bridge)",
      "baseUrl": "https://coder.example.com/api/v2/aibridge/anthropic",
      "apiKey": "<your-anthropic-api-key>",
      "provider": "anthropic",
      "maxOutputTokens": 8192,
      "extraHeaders": {
        "X-Neural Inverse Cloud-AI-Governance-Token": "<your-coder-api-token>"
      }
    },
    {
      "model": "gpt-5.2-codex",
      "displayName": "GPT (Neural Inverse Cloud AI Bridge)",
      "baseUrl": "https://coder.example.com/api/v2/aibridge/openai/v1",
      "apiKey": "<your-openai-api-key>",
      "provider": "openai",
      "maxOutputTokens": 16384,
      "extraHeaders": {
        "X-Neural Inverse Cloud-AI-Governance-Token": "<your-coder-api-token>"
      }
    }
  ]
}
```

**References:** [Factory BYOK OpenAI & Anthropic](https://docs.factory.ai/cli/byok/openai-anthropic)
