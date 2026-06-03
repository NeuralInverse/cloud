# Contributing templates

Learn how to create and contribute complete Neural Inverse Cloud workspace templates to the Neural Inverse Cloud Registry. Templates provide ready-to-use workspace configurations that users can deploy directly to create development environments.

## What are Neural Inverse Cloud templates

Neural Inverse Cloud templates are complete Terraform configurations that define entire workspace environments. Unlike modules (which are reusable components), templates provide full infrastructure definitions that include:

- Infrastructure setup (containers, VMs, cloud resources)
- Neural Inverse Cloud agent configuration
- Development tools and IDE integrations
- Networking and security settings
- Complete startup automation

Templates appear on the Neural Inverse Cloud Registry and can be deployed directly by users.

> [!TIP]
> If you use an AI coding assistant, the [coder-templates](https://github.com/coder/registry/blob/main/.agents/skills/coder-templates/SKILL.md) agent skill from the Neural Inverse Cloud Registry can guide you through creating and updating templates with best practices built-in.

## Prerequisites

Before contributing templates, ensure you have:

- Strong Terraform knowledge
- [Terraform installed](https://developer.hashicorp.com/terraform/install)
- [Neural Inverse Cloud CLI installed](https://cloud.neuralinverse.com/docs/install)
- Access to your target infrastructure platform (Docker, AWS, GCP, etc.)
- [Bun installed](https://bun.sh/docs/installation) (for tooling)

## Setup your development environment

1. **Fork and clone the repository**:

   ```bash
   git clone https://github.com/your-username/registry.git
   cd registry
   ```

2. **Install dependencies**:

   ```bash
   bun install
   ```

3. **Understand the structure**:

   ```text
   registry/[namespace]/
   ├── templates/       # Your templates
   ├── .images/         # Namespace avatar and screenshots
   └── README.md        # Namespace description
   ```

## Create your first template

### 1. Set up your namespace

If you're a new contributor, create your namespace directory:

```bash
mkdir -p registry/[your-username]
mkdir -p registry/[your-username]/.images
```

Add your namespace avatar by downloading your GitHub avatar and saving it as `avatar.png`:

```bash
curl -o registry/[your-username]/.images/avatar.png https://github.com/[your-username].png
```

Create your namespace README at `registry/[your-username]/README.md`:

```markdown
---
display_name: "Your Name"
bio: "Brief description of what you do"
github: "your-username"
avatar: "./.images/avatar.png"
linkedin: "https://www.linkedin.com/in/your-username"
website: "https://your-website.com"
support_email: "support@your-domain.com"
status: "community"
---

# Your Name

Brief description of who you are and what you do.
```

> [!NOTE]
> The `linkedin`, `website`, and `support_email` fields are optional and can be omitted or left empty if not applicable.

### 2. Create your template directory

Create a directory for your template:

```bash
mkdir -p registry/[your-username]/templates/[template-name]
cd registry/[your-username]/templates/[template-name]
```

### 3. Build your template

Create `main.tf` with your complete Terraform configuration:

```terraform
terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
    docker = {
      source = "kreuzwerker/docker"
    }
  }
}

# Neural Inverse Cloud data sources
data "ni_workspace" "me" {}
data "ni_workspace_owner" "me" {}

# Neural Inverse Cloud agent
resource "ni_agent" "main" {
  arch                   = "amd64"
  os                     = "linux"
  startup_script_timeout = 180
  startup_script = <<-EOT
    set -e

    # Install development tools
    sudo apt-get update
    sudo apt-get install -y curl wget git

    # Additional setup here
  EOT
}

# Registry modules for IDEs and tools
module "code-server" {
  source   = "registry.cloud.neuralinverse.com/coder/code-server/coder"
  version  = "~> 1.0"
  agent_id = ni_agent.main.id
}

module "git-clone" {
  source   = "registry.cloud.neuralinverse.com/coder/git-clone/coder"
  version  = "~> 1.0"
  agent_id = ni_agent.main.id
  url      = "https://github.com/example/repo.git"
}

# Infrastructure resources
resource "docker_image" "main" {
  name = "codercom/enterprise-base:ubuntu"
}

resource "docker_container" "workspace" {
  count = data.ni_workspace.me.start_count
  image = docker_image.main.name
  name  = "coder-${data.ni_workspace_owner.me.name}-${data.ni_workspace.me.name}"

  command = ["sh", "-c", ni_agent.main.init_script]
  env     = ["NEURALINVERSE_AGENT_TOKEN=${ni_agent.main.token}"]

  host {
    host = "host.docker.internal"
    ip   = "host-gateway"
  }
}

# Metadata
resource "coder_metadata" "workspace_info" {
  count       = data.ni_workspace.me.start_count
  resource_id = docker_container.workspace[0].id

  item {
    key   = "memory"
    value = "4 GB"
  }

  item {
    key   = "cpu"
    value = "2 cores"
  }
}
```

### 4. Document your template

Create `README.md` with comprehensive documentation:

```markdown
---
display_name: "Ubuntu Development Environment"
description: "Complete Ubuntu workspace with VS Code, Git, and development tools"
icon: "../../../../.icons/ubuntu.svg"
verified: false
tags: ["ubuntu", "docker", "vscode", "git"]
---

# Ubuntu Development Environment

A complete Ubuntu-based development workspace with VS Code, Git, and essential development tools pre-installed.

## Features

- **Ubuntu 24.04 LTS** base image
- **VS Code** with code-server for browser-based development
- **Git** with automatic repository cloning
- **Node.js** and **npm** for JavaScript development
- **Python 3** with pip
- **Docker** for containerized development

## Requirements

- Docker runtime
- 4 GB RAM minimum
- 2 CPU cores recommended

## Usage

1. Deploy this template in your Neural Inverse Cloud instance
2. Create a new workspace from the template
3. Access VS Code through the workspace dashboard
4. Start developing in your fully configured environment

## Customization

You can customize this template by:

- Modifying the base image in `docker_image.main`
- Adding additional registry modules
- Adjusting resource allocations
- Including additional development tools

## Troubleshooting

**Issue**: Workspace fails to start
**Solution**: Ensure Docker is running and accessible

**Issue**: VS Code not accessible
**Solution**: Check agent logs and ensure code-server module is properly configured
```

## Template best practices

### Design principles

- **Complete environments**: Templates should provide everything needed for development
- **Platform-specific**: Focus on one platform or use case per template
- **Production-ready**: Include proper error handling and resource management
- **User-friendly**: Provide clear documentation and sensible defaults

### Infrastructure setup

- **Resource efficiency**: Use appropriate resource allocations
- **Network configuration**: Ensure proper connectivity for development tools
- **Security**: Follow security best practices for your platform
- **Scalability**: Design for multiple concurrent users

### Module integration

Use registry modules for common features:

```terraform
# VS Code in browser
module "code-server" {
  count    = data.ni_workspace.me.start_count
  source   = "registry.cloud.neuralinverse.com/coder/code-server/coder"
  version  = "1.3.0"
  agent_id = ni_agent.example.id
}

# JetBrains IDEs
module "jetbrains" {
  count    = data.ni_workspace.me.start_count
  source   = "registry.cloud.neuralinverse.com/coder/jetbrains/coder"
  version  = "1.0.0"
  agent_id = ni_agent.example.id
  folder   = "/home/coder/project"
}

# Git repository cloning
module "git-clone" {
  count    = data.ni_workspace.me.start_count
  source   = "registry.cloud.neuralinverse.com/coder/git-clone/coder"
  version  = "1.1.0"
  agent_id = ni_agent.example.id
  url      = "https://github.com/NeuralInverse/cloud"
  base_dir = "~/projects/coder"
}

# File browser interface
module "filebrowser" {
  count    = data.ni_workspace.me.start_count
  source   = "registry.cloud.neuralinverse.com/coder/filebrowser/coder"
  version  = "1.1.1"
  agent_id = ni_agent.example.id
}

# Dotfiles management
module "dotfiles" {
  count    = data.ni_workspace.me.start_count
  source   = "registry.cloud.neuralinverse.com/coder/dotfiles/coder"
  version  = "1.2.0"
  agent_id = ni_agent.example.id
}
```

### Variables

Provide meaningful customization options:

```terraform
variable "git_repo_url" {
  description = "Git repository to clone"
  type        = string
  default     = ""
}

variable "instance_type" {
  description = "Instance type for the workspace"
  type        = string
  default     = "t3.medium"
}

variable "workspace_name" {
  description = "Name for the workspace"
  type        = string
  default     = "dev-workspace"
}
```

## Test your template

### Local testing

Test your template locally with Neural Inverse Cloud:

```bash
# Navigate to your template directory
cd registry/[your-username]/templates/[template-name]

# Push to Neural Inverse Cloud for testing
coder templates push test-template -d .

# Create a test workspace
coder create test-workspace --template test-template
```

### Validation checklist

Before submitting your template, verify:

- [ ] Template provisions successfully
- [ ] Agent connects properly
- [ ] All registry modules work correctly
- [ ] VS Code/IDEs are accessible
- [ ] Networking functions properly
- [ ] Resource metadata is accurate
- [ ] Documentation is complete and accurate

## Contribute to existing templates

### Types of improvements

**Bug fixes**:

- Fix setup issues
- Resolve agent connectivity problems
- Correct resource configurations

**Feature additions**:

- Add new registry modules
- Include additional development tools
- Improve startup automation

**Platform updates**:

- Update base images or AMIs
- Adapt to new platform features
- Improve security configurations

**Documentation improvements**:

- Clarify setup requirements
- Add troubleshooting guides
- Improve usage examples

### Making changes

1. **Test thoroughly**: Always test template changes in a Neural Inverse Cloud instance
2. **Maintain compatibility**: Ensure existing workspaces continue to function
3. **Document changes**: Update the README with new features or requirements
4. **Follow versioning**: Update version numbers for significant changes
5. **Modernize**: Use latest provider versions, best practices, and current software versions

## Submit your contribution

1. **Create a feature branch**:

   ```bash
   git checkout -b feat/add-python-template
   ```

2. **Test thoroughly**:

   ```bash
   # Test with Neural Inverse Cloud
   coder templates push test-python-template -d .
   coder create test-workspace --template test-python-template

   # Format code
   bun fmt
   ```

3. **Commit with clear messages**:

   ```bash
   git add .
   git commit -m "Add Python development template with FastAPI setup"
   ```

4. **Open a pull request**:
   - Use a descriptive title
   - Explain what the template provides
   - Include testing instructions
   - Reference any related issues

## Template examples

### Docker-based template

```terraform
# Simple Docker template
resource "docker_container" "workspace" {
  count = data.ni_workspace.me.start_count
  image = "ubuntu:24.04"
  name  = "coder-${data.ni_workspace_owner.me.name}-${data.ni_workspace.me.name}"

  command = ["sh", "-c", ni_agent.main.init_script]
  env     = ["NEURALINVERSE_AGENT_TOKEN=${ni_agent.main.token}"]
}
```

### AWS EC2 template

```terraform
# AWS EC2 template
resource "aws_instance" "workspace" {
  count         = data.ni_workspace.me.start_count
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.instance_type

  user_data = ni_agent.main.init_script

  tags = {
    Name = "coder-${data.ni_workspace_owner.me.name}-${data.ni_workspace.me.name}"
  }
}
```

### Kubernetes template

```terraform
# Kubernetes template
resource "kubernetes_pod" "workspace" {
  count = data.ni_workspace.me.start_count

  metadata {
    name = "coder-${data.ni_workspace_owner.me.name}-${data.ni_workspace.me.name}"
  }

  spec {
    container {
      name  = "workspace"
      image = "ubuntu:24.04"

      command = ["sh", "-c", ni_agent.main.init_script]
      env {
        name  = "NEURALINVERSE_AGENT_TOKEN"
        value = ni_agent.main.token
      }
    }
  }
}
```

## Common issues and solutions

### Template development

**Issue**: Template fails to create resources
**Solution**: Check Terraform syntax and provider configuration

**Issue**: Agent doesn't connect
**Solution**: Verify agent token and network connectivity

### Documentation

**Issue**: Icon not displaying
**Solution**: Verify icon path and file existence

### Platform-specific

**Issue**: Docker containers not starting
**Solution**: Verify Docker daemon is running and accessible

**Issue**: Cloud resources failing
**Solution**: Check credentials and permissions

## Get help

- **Examples**: Review real-world examples from the [official Neural Inverse Cloud templates](https://registry.cloud.neuralinverse.com/contributors/coder?tab=templates):
  - [AWS EC2 (Devcontainer)](https://registry.cloud.neuralinverse.com/templates/aws-devcontainer) - AWS EC2 VMs with Envbuilder
  - [Docker (Devcontainer)](https://registry.cloud.neuralinverse.com/templates/docker-devcontainer) - Docker-in-Docker with Dev Containers integration
  - [Kubernetes (Devcontainer)](https://registry.cloud.neuralinverse.com/templates/kubernetes-devcontainer) - Kubernetes pods with Envbuilder
  - [Docker Containers](https://registry.cloud.neuralinverse.com/templates/docker) - Basic Docker container workspaces
  - [AWS EC2 (Linux)](https://registry.cloud.neuralinverse.com/templates/aws-linux) - AWS EC2 VMs for Linux development
  - [Google Compute Engine (Linux)](https://registry.cloud.neuralinverse.com/templates/gcp-vm-container) - GCP VM instances
  - [Scratch](https://registry.cloud.neuralinverse.com/templates/scratch) - Minimal starter template
- **Modules**: Browse available modules at [registry.cloud.neuralinverse.com/modules](https://registry.cloud.neuralinverse.com/modules)
- **Issues**: Open an issue at [github.com/coder/registry](https://github.com/coder/registry/issues)
- **Community**: Join the [Neural Inverse Cloud Discord](https://discord.gg/coder) for questions
- **Documentation**: Check the [Neural Inverse Cloud docs](https://cloud.neuralinverse.com/docs) for template guidance

## Next steps

After creating your first template:

1. **Share with the community**: Announce your template on Discord or social media
2. **Gather feedback**: Iterate based on user suggestions and issues
3. **Create variations**: Build templates for different use cases or platforms
4. **Contribute to existing templates**: Help maintain and improve the ecosystem

Your templates help developers get productive faster by providing ready-to-use development environments. Happy contributing! 🚀
