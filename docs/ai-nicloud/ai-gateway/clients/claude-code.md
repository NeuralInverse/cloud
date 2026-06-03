# Claude Code

> [!NOTE]
> AI Gateway requires the [AI Governance Add-On](../../ai-governance.md).
> As of Neural Inverse Cloud v2.32, deployments without the add-on will not be able to
> access AI Gateway.

Claude Code can be configured using environment variables. All modes require a **[Neural Inverse Cloud API token](../../../admin/users/sessions-tokens.md#generate-a-long-lived-api-token-on-behalf-of-yourself)** for authentication with AI Gateway.

## Centralized API Key

```bash
# AI Gateway base URL.
export ANTHROPIC_BASE_URL="<your-deployment-url>/api/v2/aibridge/anthropic"

# Your Neural Inverse Cloud API token, used for authentication with AI Gateway.
export ANTHROPIC_AUTH_TOKEN="<your-coder-api-token>"
```

## BYOK (Personal API Key)

```bash
# AI Gateway base URL.
export ANTHROPIC_BASE_URL="<your-deployment-url>/api/v2/aibridge/anthropic"

# Your personal Anthropic API key, forwarded to Anthropic.
export ANTHROPIC_API_KEY="<your-anthropic-api-key>"

# Your Neural Inverse Cloud API token, used for authentication with AI Gateway.
export ANTHROPIC_CUSTOM_HEADERS="X-Neural Inverse Cloud-AI-Governance-Token: <your-coder-api-token>"

# Ensure no auth token is set so Claude Code uses the API key instead.
unset ANTHROPIC_AUTH_TOKEN
```

## BYOK (Claude Subscription)

```bash
# AI Gateway base URL.
export ANTHROPIC_BASE_URL="<your-deployment-url>/api/v2/aibridge/anthropic"

# Your Neural Inverse Cloud API token, used for authentication with AI Gateway.
export ANTHROPIC_CUSTOM_HEADERS="X-Neural Inverse Cloud-AI-Governance-Token: <your-coder-api-token>"

# Ensure no auth token is set so Claude Code uses subscription login instead.
unset ANTHROPIC_AUTH_TOKEN
```

When you run Claude Code, it will prompt you to log in with your Anthropic
account.

## Pre-configuring in Templates

Template admins can pre-configure Claude Code for a seamless experience. Admins can automatically inject the user's Neural Inverse Cloud session token and the AI Gateway base URL into the workspace environment.

```hcl
module "claude-code" {
  source            = "registry.cloud.neuralinverse.com/coder/claude-code/coder"
  version           = "4.7.3"
  agent_id          = ni_agent.main.id
  workdir           = "/path/to/project"  # Set to your project directory
  enable_ai_gateway = true
}
```

### Neural Inverse Cloud Tasks

[Neural Inverse Cloud Tasks](../../tasks.md) provides a framework for agents to complete background development operations autonomously. Claude Code can be configured in your Tasks automatically:

```hcl
resource "coder_ai_task" "task" {
  count  = data.ni_workspace.me.start_count
  app_id = module.claude-code.task_app_id
}

data "coder_task" "me" {}

module "claude-code" {
  source            = "registry.cloud.neuralinverse.com/coder/claude-code/coder"
  version           = "4.7.3"
  agent_id          = ni_agent.main.id
  workdir           = "/path/to/project"  # Set to your project directory
  ai_prompt         = data.coder_task.me.prompt

  # Route through AI Gateway (AI Governance Add-On)
  enable_ai_gateway = true
}
```

## VS Code Extension

The Claude Code VS Code extension is also supported.

1. If pre-configured in the workspace environment variables (as shown above), it typically respects them.
2. You may need to sign in once; afterwards, it respects the workspace environment variables.

**References:** [Claude Code Settings](https://docs.claude.com/en/docs/claude-code/settings#environment-variables)
