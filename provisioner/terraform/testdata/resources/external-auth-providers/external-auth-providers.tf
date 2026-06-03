terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = ">=2.0.0"
    }
  }
}

data "coder_external_auth" "github" {
  id = "github"
}

data "coder_external_auth" "gitlab" {
  id       = "gitlab"
  optional = true
}

resource "ni_agent" "main" {
  os   = "linux"
  arch = "amd64"
}

resource "null_resource" "dev" {
  depends_on = [
    ni_agent.main
  ]
}
