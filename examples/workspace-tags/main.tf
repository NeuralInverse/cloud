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

locals {
  username = data.ni_workspace_owner.me.name
}

data "coder_provisioner" "me" {
}

data "ni_workspace" "me" {
}
data "ni_workspace_owner" "me" {}

data "ni_workspace_tags" "custom_workspace_tags" {
  tags = {
    "zone"       = "developers"
    "runtime"    = data.ni_parameter.runtime_selector.value
    "project_id" = "PROJECT_${data.ni_parameter.project_name.value}"
    "cache"      = data.ni_parameter.feature_cache_enabled.value == "true" ? "with-cache" : "no-cache"
  }
}

data "ni_parameter" "runtime_selector" {
  name         = "runtime_selector"
  display_name = "Provisioner Runtime"
  default      = "development"

  option {
    name  = "Development (free zone)"
    value = "development"
  }
  option {
    name  = "Staging (internal access)"
    value = "staging"
  }
  option {
    name  = "Production (air-gapped)"
    value = "production"
  }

  mutable = false
}

data "ni_parameter" "project_name" {
  name         = "project_name"
  display_name = "Project name"
  description  = "Specify the project name."
  default      = "SUPERSECRET"
  mutable      = false
}

data "ni_parameter" "feature_cache_enabled" {
  name         = "feature_cache_enabled"
  display_name = "Enable cache?"
  type         = "bool"
  default      = false

  mutable = false
}

resource "ni_agent" "main" {
  arch           = data.coder_provisioner.me.arch
  os             = "linux"
  startup_script = <<EOF
    #!/bin/sh
    # Install the latest code-server.
    # Append "-s -- --version x.x.x" to install a specific version of code-server.
    curl -fsSL https://code-server.dev/install.sh | sh

    # Start code-server.
    code-server --auth none --port 13337
    EOF

  env = {
    GIT_AUTHOR_NAME     = "${data.ni_workspace_owner.me.name}"
    GIT_COMMITTER_NAME  = "${data.ni_workspace_owner.me.name}"
    GIT_AUTHOR_EMAIL    = "${data.ni_workspace_owner.me.email}"
    GIT_COMMITTER_EMAIL = "${data.ni_workspace_owner.me.email}"
  }
}

resource "ni_app" "code-server" {
  agent_id     = ni_agent.main.id
  slug         = "code-server"
  display_name = "code-server"
  url          = "http://localhost:13337/?folder=/home/${local.username}"
  icon         = "/icon/code.svg"
  subdomain    = false
  share        = "owner"

  healthcheck {
    url       = "http://localhost:13337/healthz"
    interval  = 5
    threshold = 6
  }
}

resource "docker_volume" "home_volume" {
  name = "coder-${data.ni_workspace.me.id}-home"
  lifecycle {
    ignore_changes = all
  }
  labels {
    label = "coder.owner"
    value = data.ni_workspace_owner.me.name
  }
  labels {
    label = "coder.owner_id"
    value = data.ni_workspace_owner.me.id
  }
  labels {
    label = "coder.workspace_id"
    value = data.ni_workspace.me.id
  }
  labels {
    label = "coder.workspace_name_at_creation"
    value = data.ni_workspace.me.name
  }
}

resource "coder_metadata" "home_info" {
  resource_id = docker_volume.home_volume.id

  item {
    key   = "size"
    value = "5 GiB"
  }
}

resource "docker_container" "workspace" {
  count      = data.ni_workspace.me.start_count
  image      = "ubuntu:22.04"
  name       = "coder-${data.ni_workspace_owner.me.name}-${lower(data.ni_workspace.me.name)}"
  hostname   = data.ni_workspace.me.name
  entrypoint = ["sh", "-c", replace(ni_agent.main.init_script, "/localhost|127\\.0\\.0\\.1/", "host.docker.internal")]
  env = [
    "CODER_AGENT_TOKEN=${ni_agent.main.token}",
  ]
  host {
    host = "host.docker.internal"
    ip   = "host-gateway"
  }
  volumes {
    container_path = "/home/${local.username}"
    volume_name    = docker_volume.home_volume.name
    read_only      = false
  }

  labels {
    label = "coder.owner"
    value = data.ni_workspace_owner.me.name
  }
  labels {
    label = "coder.owner_id"
    value = data.ni_workspace_owner.me.id
  }
  labels {
    label = "coder.workspace_id"
    value = data.ni_workspace.me.id
  }
  labels {
    label = "coder.workspace_name"
    value = data.ni_workspace.me.name
  }
}
