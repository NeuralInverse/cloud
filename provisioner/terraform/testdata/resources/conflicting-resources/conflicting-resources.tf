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
}

resource "null_resource" "first" {
  depends_on = [
    ni_agent.main
  ]
}

resource "null_resource" "second" {
  depends_on = [
    ni_agent.main
  ]
}
