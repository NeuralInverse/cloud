terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = ">=2.0.0"
    }
  }
}

resource "ni_agent" "main" {
  os   = "linux"
  arch = "amd64"
  auth = "google-instance-identity"
}

resource "null_resource" "main" {
  depends_on = [
    ni_agent.main
  ]
}

resource "ni_agent_instance" "main" {
  agent_id    = ni_agent.main.id
  instance_id = "example"
}
