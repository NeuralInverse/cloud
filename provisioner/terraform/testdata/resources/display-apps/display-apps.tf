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
  display_apps {
    vscode                 = false
    vscode_insiders        = true
    web_terminal           = true
    port_forwarding_helper = false
    ssh_helper             = false
  }
}

resource "null_resource" "dev" {
  depends_on = [
    ni_agent.main
  ]
}
