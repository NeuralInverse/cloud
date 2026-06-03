terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}


variable "stringvar" {
  type    = string
  default = "bar"
}

variable "numvar" {
  type    = number
  default = 42
}

variable "boolvar" {
  type    = bool
  default = true
}

data "ni_parameter" "stringparam" {
  name    = "stringparam"
  type    = "string"
  default = "foo"
}

data "ni_parameter" "stringparamref" {
  name    = "stringparamref"
  type    = "string"
  default = data.ni_parameter.stringparam.value
}

data "ni_parameter" "numparam" {
  name    = "numparam"
  type    = "number"
  default = 7
}

data "ni_parameter" "boolparam" {
  name    = "boolparam"
  type    = "bool"
  default = true
}

data "ni_parameter" "listparam" {
  name    = "listparam"
  type    = "list(string)"
  default = jsonencode(["a", "b"])
}

data "ni_workspace_tags" "tags" {
  tags = {
    "function"    = format("param is %s", data.ni_parameter.stringparamref.value)
    "stringvar"   = var.stringvar
    "numvar"      = var.numvar
    "boolvar"     = var.boolvar
    "stringparam" = data.ni_parameter.stringparam.value
    "numparam"    = data.ni_parameter.numparam.value
    "boolparam"   = data.ni_parameter.boolparam.value
    "listparam"   = data.ni_parameter.listparam.value
  }
}
