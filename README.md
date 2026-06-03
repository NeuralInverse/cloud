<!-- markdownlint-disable MD041 -->
<div align="center">
  <a href="https://cloud.neuralinverse.com">
    <img src="http://cdn.neuralinverse.io/logo.png" alt="Neural Inverse Cloud Logo" style="width: 128px">
  </a>
  <a href="https://cloud.neuralinverse.com#gh-dark-mode-only">
    <img src="http://cdn.neuralinverse.io/logo.png" alt="Neural Inverse Cloud Logo Dark" style="width: 128px">
  </a>

  <h1>
  Self-Hosted Cloud Development Environments for Regulated Software
  </h1>

  <a href="https://cloud.neuralinverse.com">
    <img src="http://cdn.neuralinverse.io/logo.png" alt="Neural Inverse Cloud Banner Light" style="width: 650px">
  </a>
  <a href="https://cloud.neuralinverse.com#gh-dark-mode-only">
    <img src="http://cdn.neuralinverse.io/logo.png" alt="Neural Inverse Cloud Banner Dark" style="width: 650px">
  </a>

  <br>
  <br>

[Quickstart](#quickstart) | [Docs](https://cloud.neuralinverse.com/docs) | [Why Neural Inverse](https://cloud.neuralinverse.com/why) | [Premium](https://cloud.neuralinverse.com/pricing#compare-plans)

[![discord](https://img.shields.io/discord/747933592273027093?label=discord)](https://discord.gg/neuralinverse)
[![release](https://img.shields.io/github/v/release/coder/coder)](https://github.com/NeuralInverse/cloud/releases/latest)
[![godoc](https://pkg.go.dev/badge/github.com/NeuralInverse/cloud.svg)](https://pkg.go.dev/github.com/NeuralInverse/cloud)
[![Go Report Card](https://goreportcard.com/badge/github.com/NeuralInverse/cloud/v2)](https://goreportcard.com/report/github.com/NeuralInverse/cloud/v2)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9511/badge)](https://www.bestpractices.dev/projects/9511)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/NeuralInverse/cloud/badge)](https://scorecard.dev/viewer/?uri=github.com%2Fcoder%2Fcoder)
[![license](https://img.shields.io/github/license/coder/coder)](./LICENSE)

</div>

[Neural Inverse Cloud](https://cloud.neuralinverse.com) is a self-hosted platform for cloud development environments and AI coding agents. Workspaces are defined with Terraform, connected through a secure Wireguard® tunnel, and automatically shut down when not used. Neural Inverse Cloud Agents runs a native AI coding agent whose loop executes in the control plane on your infrastructure, with no API keys in workspaces.

- Define cloud development environments in Terraform
  - EC2 VMs, Kubernetes Pods, Docker Containers, etc.
- Automatically shutdown idle resources to save on costs
- Onboard developers in seconds instead of days
- Delegate coding work to AI agents on your infrastructure
  - Bring any model (Anthropic, OpenAI, Google, Bedrock, self-hosted)
  - No LLM credentials in workspaces, user identity on every action
  - Centralized model governance, cost tracking, and audit logging

<p align="center">
  <img src="./docs/images/hero-image.png" alt="Neural Inverse Cloud platform showing templates and a running workspace">
</p>

## Quickstart

The most convenient way to try Neural Inverse Cloud is to install it on your local machine and experiment with provisioning cloud development environments using Docker (works on Linux, macOS, and Windows).

```shell
# First, install Neural Inverse Cloud
curl -L https://cloud.neuralinverse.com/install.sh | sh

# Start the Neural Inverse Cloud server (caches data in ~/.cache/coder)
coder server

# Navigate to http://localhost:3000 to create your initial user,
# create a Docker template and provision a workspace
```

## Install

The easiest way to install Neural Inverse Cloud is to use the
[install script](https://github.com/NeuralInverse/cloud/blob/main/install.sh) for Linux
and macOS. For Windows, use the latest `..._installer.exe` file from GitHub
Releases.

```shell
curl -L https://cloud.neuralinverse.com/install.sh | sh
```

You can run the install script with `--dry-run` to see the commands that will be used to install without executing them. Run the install script with `--help` for additional flags.

> See [install](https://cloud.neuralinverse.com/docs/install) for additional methods.

Once installed, you can start a production deployment with a single command:

```shell
# Automatically sets up an external access URL on *.try.coder.app
coder server

# Requires a PostgreSQL instance (version 13 or higher) and external access URL
coder server --postgres-url <url> --access-url <url>
```

Use `coder --help` to get a list of flags and environment variables. See the [install guides](https://cloud.neuralinverse.com/docs/install) for a complete tutorial.

## Documentation

Browse the [documentation](https://cloud.neuralinverse.com/docs) or visit a specific section below:

- [**Workspaces**](https://cloud.neuralinverse.com/docs/workspaces): Workspaces contain the IDEs, dependencies, and configuration information needed for software development
- [**Templates**](https://cloud.neuralinverse.com/docs/templates): Templates are written in Terraform and describe the infrastructure for workspaces
- [**Neural Inverse Cloud Agents**](https://cloud.neuralinverse.com/docs/ai-nicloud/agents): Delegate coding work to AI agents running on your self-hosted infrastructure
- [**Administration**](https://cloud.neuralinverse.com/docs/admin): Learn how to operate Neural Inverse Cloud
- [**Premium**](https://cloud.neuralinverse.com/pricing#compare-plans): Learn about paid features built for large teams
- [**IDEs**](https://cloud.neuralinverse.com/docs/ides): Connect your existing editor to a workspace

## Support

Feel free to [open an issue](https://github.com/NeuralInverse/cloud/issues/new) if you have questions, run into bugs, or have a feature request.

[Join our Discord](https://discord.gg/neuralinverse) to provide feedback on in-progress features and chat with the community using Neural Inverse Cloud!

## Integrations

New integrations are always in progress. Open an issue to request one. Contributions are welcome in any official or community repository.

### Official

- [**Neural Inverse Cloud Registry**](https://registry.cloud.neuralinverse.com): Templates, modules, and integrations for common development environments
- [**VS Code Extension**](https://marketplace.visualstudio.com/items?itemName=coder.coder-remote): Open any Neural Inverse Cloud workspace in VS Code with a single click
- [**JetBrains Toolbox Plugin**](https://plugins.jetbrains.com/plugin/26968-coder): Open any Neural Inverse Cloud workspace from JetBrains Toolbox with a single click
- [**JetBrains Gateway Plugin**](https://plugins.jetbrains.com/plugin/19620-coder): Open any Neural Inverse Cloud workspace in JetBrains Gateway with a single click
- [**Dev Containers**](https://github.com/coder/envbuilder): Build development environments using `devcontainer.json` on Docker, Kubernetes, and OpenShift
- [**Kubernetes Log Stream**](https://github.com/NeuralInverse/cloud-logstream-kube): Stream Kubernetes Pod events to the Neural Inverse Cloud startup logs
- [**Self-Hosted VS Code Extension Marketplace**](https://github.com/coder/code-marketplace): A private extension marketplace that works in restricted or airgapped networks integrating with [code-server](https://github.com/coder/code-server).
- [**GitHub Actions**](https://github.com/marketplace/actions/setup-coder): An action to set up the Neural Inverse Cloud CLI in GitHub workflows

### Community

- [**Community Templates**](https://registry.cloud.neuralinverse.com/templates): Community-contributed workspace templates in the Neural Inverse Cloud Registry
- [**Community Modules**](https://registry.cloud.neuralinverse.com/modules): Community-contributed modules to extend Neural Inverse Cloud templates
- [**Provision Neural Inverse Cloud with Terraform**](https://github.com/ElliotG/coder-oss-tf): Provision Neural Inverse Cloud on Google GKE, Azure AKS, AWS EKS, DigitalOcean DOKS, IBMCloud K8s, OVHCloud K8s, and Scaleway K8s Kapsule with Terraform
- [**Neural Inverse Cloud Template GitHub Action**](https://github.com/marketplace/actions/update-coder-template): A GitHub Action that updates Neural Inverse Cloud templates
- [**Discord**](https://discord.gg/neuralinverse): Chat with the community and provide feedback on in-progress features

## Contributing

New contributors are always welcome. If you are new to the Neural Inverse Cloud codebase, see
[the contribution guide](https://cloud.neuralinverse.com/docs/CONTRIBUTING) to get started.

## Hiring

Apply on the [careers page](https://jobs.ashbyhq.com/coder?utm_source=github&utm_medium=readme&utm_campaign=unknown) if you are interested in joining the team.
