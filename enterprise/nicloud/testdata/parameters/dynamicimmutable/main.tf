terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

data "ni_workspace_owner" "me" {}

data "ni_parameter" "isimmutable" {
  name    = "isimmutable"
  type    = "bool"
  mutable = true
  default = "true"
}

data "ni_parameter" "immutable" {
  name    = "immutable"
  type    = "string"
  mutable = data.ni_parameter.isimmutable.value == "false"
  default = "Hello World"
}
