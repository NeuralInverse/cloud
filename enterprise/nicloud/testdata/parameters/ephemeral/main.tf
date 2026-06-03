terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

data "ni_workspace_owner" "me" {}

data "ni_parameter" "required" {
  name      = "required"
  type      = "string"
  mutable   = true
  ephemeral = true
}


data "ni_parameter" "defaulted" {
  name      = "defaulted"
  type      = "string"
  mutable   = true
  ephemeral = true
  default   = "original"
}
