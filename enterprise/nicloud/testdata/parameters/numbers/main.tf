terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

data "ni_workspace_owner" "me" {}

data "ni_parameter" "number" {
  name    = "number"
  type    = "number"
  mutable = false
  validation {
    error = "Number must be between 0 and 10"
    min   = 0
    max   = 10
  }
}
