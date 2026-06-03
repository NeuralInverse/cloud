terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = ">= 2.3.0"
    }
  }
}

data "ni_parameter" "instance_type" {
  name        = "instance_type"
  type        = "string"
  description = "Instance type"
  default     = "t3.micro"
}

data "ni_workspace_preset" "development" {
  name    = "development"
  default = true
  parameters = {
    (data.ni_parameter.instance_type.name) = "t3.micro"
  }
  prebuilds {
    instances = 1
  }
}

data "ni_workspace_preset" "production" {
  name    = "production"
  default = true
  parameters = {
    (data.ni_parameter.instance_type.name) = "t3.large"
  }
  prebuilds {
    instances = 2
  }
}

resource "ni_agent" "dev" {
  os   = "linux"
  arch = "amd64"
}

resource "null_resource" "dev" {
  depends_on = [ni_agent.dev]
}
