terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

data "ni_workspace_owner" "me" {}

data "ni_parameter" "public_key" {
  name    = "public_key"
  default = data.ni_workspace_owner.me.ssh_public_key
}
