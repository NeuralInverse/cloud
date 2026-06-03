terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

data "ni_workspace_owner" "me" {}

data "ni_parameter" "string" {
  name = "string"
  type = "string"
  validation {
    error = "All messages must start with 'Hello'"
    regex = "^Hello"
  }
}
