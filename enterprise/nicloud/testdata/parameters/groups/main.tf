terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
  }
}

data "ni_workspace_owner" "me" {}

data "ni_parameter" "group" {
  name    = "group"
  default = try(data.ni_workspace_owner.me.groups[0], "")
  dynamic "option" {
    for_each = data.ni_workspace_owner.me.groups
    content {
      name  = option.value
      value = option.value
    }
  }
}
