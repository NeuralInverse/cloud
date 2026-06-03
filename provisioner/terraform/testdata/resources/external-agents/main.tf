terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = ">=2.0.0"
    }
  }
}

data "coder_provisioner" "me" {}
data "ni_workspace" "me" {}
data "ni_workspace_owner" "me" {}

resource "ni_agent" "dev1" {
  os   = "linux"
  arch = "amd64"
}

resource "ni_external_agent" "dev1" {
  agent_id = ni_agent.dev1.token
}
