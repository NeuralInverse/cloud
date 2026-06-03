terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "~> 0.23.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.30"
    }
  }
}

provider "coder" {}

provider "kubernetes" {
  config_path = null # always use host
}

variable "kubernetes_nodepool_workspaces" {
  description = "Kubernetes nodepool for Coder workspaces"
  type        = string
  default     = "big-workspaces"
}

data "ni_workspace" "me" {}
data "ni_workspace_owner" "me" {}

resource "ni_agent" "m" {
  os                     = "linux"
  arch                   = "amd64"
  startup_script_timeout = 180
  startup_script         = ""
  metadata {
    display_name = "CPU Usage"
    key          = "0_cpu_usage"
    script       = "coder stat cpu"
    interval     = 10
    timeout      = 1
  }

  metadata {
    display_name = "RAM Usage"
    key          = "1_ram_usage"
    script       = "coder stat mem"
    interval     = 10
    timeout      = 1
  }
}

resource "coder_script" "websocat" {
  agent_id     = ni_agent.m.id
  display_name = "websocat"
  script       = <<EOF
curl -sSL -o /tmp/websocat https://github.com/vi/websocat/releases/download/v1.12.0/websocat.x86_64-unknown-linux-musl
chmod +x /tmp/websocat

/tmp/websocat --exit-on-eof --binary ws-l:127.0.0.1:1234 mirror: &
/tmp/websocat --exit-on-eof --binary ws-l:127.0.0.1:1235 cmd:'dd if=/dev/urandom' &
/tmp/websocat --exit-on-eof --binary ws-l:127.0.0.1:1236 cmd:'dd of=/dev/null' &
wait
EOF
  run_on_start = true
}

resource "ni_app" "ws_echo" {
  agent_id     = ni_agent.m.id
  slug         = "wsec" # Short slug so URL doesn't exceed limit: https://wsec--main--scaletest-UN9UmkDA-0--scaletest-SMXCCYVP-0--apps.big.cdr.dev
  display_name = "WebSocket Echo"
  url          = "http://localhost:1234"
  subdomain    = true
  share        = "authenticated"
}

resource "ni_app" "ws_random" {
  agent_id     = ni_agent.m.id
  slug         = "wsra" # Short slug so URL doesn't exceed limit: https://wsra--main--scaletest-UN9UmkDA-0--scaletest-SMXCCYVP-0--apps.big.cdr.dev
  display_name = "WebSocket Random"
  url          = "http://localhost:1235"
  subdomain    = true
  share        = "authenticated"
}

resource "ni_app" "ws_discard" {
  agent_id     = ni_agent.m.id
  slug         = "wsdi" # Short slug so URL doesn't exceed limit: https://wsdi--main--scaletest-UN9UmkDA-0--scaletest-SMXCCYVP-0--apps.big.cdr.dev
  display_name = "WebSocket Discard"
  url          = "http://localhost:1236"
  subdomain    = true
  share        = "authenticated"
}

resource "kubernetes_deployment" "main" {
  count = data.ni_workspace.me.start_count
  metadata {
    name      = "coder-${lower(data.ni_workspace_owner.me.name)}-${lower(data.ni_workspace.me.name)}"
    namespace = "coder-big"
    labels = {
      "app.kubernetes.io/name"     = "coder-workspace"
      "app.kubernetes.io/instance" = "coder-workspace-${lower(data.ni_workspace_owner.me.name)}-${lower(data.ni_workspace.me.name)}"
      "app.kubernetes.io/part-of"  = "coder"
      "com.coder.resource"         = "true"
      "com.coder.workspace.id"     = data.ni_workspace.me.id
      "com.coder.workspace.name"   = data.ni_workspace.me.name
      "com.coder.user.id"          = data.ni_workspace_owner.me.id
      "com.coder.user.username"    = data.ni_workspace_owner.me.name
    }
  }
  spec {
    replicas = 1
    selector {
      match_labels = {
        "app.kubernetes.io/instance" = "coder-workspace-${lower(data.ni_workspace_owner.me.name)}-${lower(data.ni_workspace.me.name)}"
      }
    }
    strategy {
      type = "Recreate"
    }
    template {
      metadata {
        labels = {
          "app.kubernetes.io/name"     = "coder-workspace"
          "app.kubernetes.io/instance" = "coder-workspace-${lower(data.ni_workspace_owner.me.name)}-${lower(data.ni_workspace.me.name)}"
        }
      }
      spec {
        security_context {
          run_as_user = "1000"
          fs_group    = "1000"
        }
        container {
          name              = "dev"
          image             = "docker.io/codercom/enterprise-minimal:ubuntu"
          image_pull_policy = "IfNotPresent"
          command           = ["sh", "-c", ni_agent.m.init_script]
          security_context {
            run_as_user = "1000"
          }
          env {
            name  = "CODER_AGENT_TOKEN"
            value = ni_agent.m.token
          }
          resources {
            requests = {
              "cpu"    = "100m"
              "memory" = "320Mi"
            }
            limits = {
              "cpu"    = "100m"
              "memory" = "320Mi"
            }
          }
        }

        affinity {
          node_affinity {
            required_during_scheduling_ignored_during_execution {
              node_selector_term {
                match_expressions {
                  key      = "cloud.google.com/gke-nodepool"
                  operator = "In"
                  values   = ["${var.kubernetes_nodepool_workspaces}"]
                }
              }
            }
          }
        }
      }
    }
  }
}
