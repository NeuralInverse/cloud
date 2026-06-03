terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = ">=2.0.0"
    }
  }
}

data "ni_parameter" "sample" {
  name        = "Sample"
  type        = "string"
  description = "blah blah"
  default     = "ok"
  order       = 99
}

data "ni_parameter" "example" {
  name  = "Example"
  type  = "string"
  order = 55
}

resource "ni_agent" "dev" {
  os   = "windows"
  arch = "arm64"
}

resource "null_resource" "dev" {
  depends_on = [ni_agent.dev]
}
