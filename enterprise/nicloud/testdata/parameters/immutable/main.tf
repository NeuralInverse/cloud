terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

data "ni_workspace_owner" "me" {}

data "ni_parameter" "immutable" {
  name    = "immutable"
  type    = "string"
  mutable = false
  default = "Hello World"
}
