<!-- markdownlint-disable MD041 -->
<div align="center">
  <a href="https://cloud.neuralinverse.com">
    <img src="http://cdn.neuralinverse.io/logo.png" alt="Neural Inverse Cloud" style="width: 128px">
  </a>

  <h1>Neural Inverse Cloud</h1>
  <p><strong>Self-Hosted Cloud Development Environments for Regulated Software</strong></p>

  <p>
    <a href="https://cloud.neuralinverse.com/docs">Documentation</a> &middot;
    <a href="https://discord.gg/neuralinverse">Discord</a> &middot;
    <a href="https://github.com/NeuralInverse/cloud/issues">Issues</a> &middot;
    <a href="https://neuralinverse.com">Website</a>
  </p>

  <p>
    <a href="https://github.com/NeuralInverse/cloud/releases/latest"><img src="https://img.shields.io/github/v/release/NeuralInverse/cloud" alt="Release"></a>
    <a href="./LICENSE"><img src="https://img.shields.io/github/license/NeuralInverse/cloud" alt="License"></a>
    <a href="https://discord.gg/neuralinverse"><img src="https://img.shields.io/discord/747933592273027093?label=discord" alt="Discord"></a>
  </p>
</div>

---

## What is Neural Inverse Cloud?

Neural Inverse Cloud is a self-hosted platform for provisioning secure, compliant cloud development environments and AI coding agents — built for teams in defense, energy, medical devices, and other regulated industries.

- **Compliance-first**: Built to support teams working under IEC 61508, ISO 26262, DO-178C, MISRA, and other frameworks
- **Infrastructure as Code**: Define environments in Terraform (EC2, Kubernetes, Docker, etc.)
- **AI Agents on your infra**: Run AI coding agents in the control plane — no API keys in workspaces, full audit trail
- **Secure by default**: Wireguard tunnels, auto-shutdown idle workspaces, air-gap ready
- **Any IDE**: VS Code, JetBrains, browser-based, or Neural Inverse CE

## Architecture

```
Developer -> Neural Inverse Cloud -> Workspace (Terraform-provisioned)
                |                         |
                |-- AI Agent Loop         |-- Your IDE
                |-- Audit Logging         |-- Your Code
                |-- Model Governance      |-- Compliance Tools
```

## Quickstart

```shell
# Install
curl -L https://cloud.neuralinverse.com/install.sh | sh

# Start the server
neuralinverse server

# Open http://localhost:3000
# Create a template, provision a workspace
```

## Install

```shell
# Linux / macOS
curl -L https://cloud.neuralinverse.com/install.sh | sh

# Or from source
git clone https://github.com/NeuralInverse/cloud.git
cd cloud
make build
```

For Windows, download the latest installer from [Releases](https://github.com/NeuralInverse/cloud/releases/latest).

See the full [installation guide](https://cloud.neuralinverse.com/docs/install) for production deployments.

## Production Deployment

```shell
# With external access URL (auto-TLS)
neuralinverse server --access-url https://cloud.yourcompany.com

# With PostgreSQL (recommended for production)
neuralinverse server --postgres-url postgresql://user:pass@host/db --access-url https://cloud.yourcompany.com
```

Run `neuralinverse --help` for all flags and environment variables.

## Key Features

| Feature | Description |
|---------|-------------|
| **Workspace Templates** | Terraform-defined dev environments (Docker, K8s, VMs) |
| **AI Agents** | Delegate coding tasks to AI on your infrastructure |
| **Model Governance** | Bring any model (Anthropic, OpenAI, Google, Bedrock, self-hosted) |
| **Auto-Shutdown** | Idle workspaces shut down automatically to save costs |
| **Audit Logging** | Every action logged — user identity on every AI action |
| **Air-Gap Support** | Deploy in disconnected networks |
| **SSO/SAML** | Enterprise authentication out of the box |
| **Wireguard Tunnels** | Secure connectivity to workspaces |

## Documentation

- [**Workspaces**](https://cloud.neuralinverse.com/docs/workspaces) — Dev environments with IDEs, dependencies, and config
- [**Templates**](https://cloud.neuralinverse.com/docs/templates) — Terraform definitions for workspace infrastructure
- [**AI Agents**](https://cloud.neuralinverse.com/docs/ai-agents) — Run coding agents on your self-hosted infrastructure
- [**Administration**](https://cloud.neuralinverse.com/docs/admin) — Operating Neural Inverse Cloud
- [**IDEs**](https://cloud.neuralinverse.com/docs/ides) — Connect VS Code, JetBrains, or any editor

## Integrations

- [**Neural Inverse Registry**](https://registry.cloud.neuralinverse.com) — Templates, modules, and integrations
- [**VS Code Extension**](https://marketplace.visualstudio.com/items?itemName=neuralinverse.ni-remote) — Open workspaces in VS Code
- [**JetBrains Plugin**](https://plugins.jetbrains.com/plugin/26968-neuralinverse) — Open workspaces from JetBrains
- [**Dev Containers**](https://github.com/NeuralInverse/envbuilder) — Build from `devcontainer.json`
- [**GitHub Actions**](https://github.com/marketplace/actions/setup-neuralinverse) — CI/CD integration

## Community

- [Discord](https://discord.gg/neuralinverse) — Chat, feedback, support
- [Issues](https://github.com/NeuralInverse/cloud/issues) — Bug reports and feature requests
- [Contributing](https://cloud.neuralinverse.com/docs/contributing) — New contributors welcome

## License

Neural Inverse Cloud is licensed under [AGPL-3.0](./LICENSE).

This is a fork of [Coder](https://github.com/coder/coder), rebranded and extended for regulated software development.

---

<div align="center">
  <p><strong>Built by <a href="https://neuralinverse.com">Neural Inverse</a></strong></p>
  <p>AI-Native IDE & Cloud for Regulated Software</p>
</div>
