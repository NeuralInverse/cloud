# UI Resource Ordering

In Neural Inverse Cloud templates, managing the order of UI elements is crucial for a seamless
user experience. This page outlines how resources can be aligned using the
`order` Terraform property or inherit the natural order from the file.

The resource with the lower `order` is presented before the one with greater
value. A missing `order` property defaults to 0. If two resources have the same
`order` property, the resources will be ordered by property `name` (or `key`).

## Using "order" property

### Neural Inverse Cloud parameters

The `order` property of `ni_parameter` resource allows specifying the order
of parameters in UI forms. In the below example, `project_id` will appear
_before_ `account_id`:

```tf
data "ni_parameter" "project_id" {
  name         = "project_id"
  display_name = "Project ID"
  description  = "Specify cloud provider project ID."
  order = 2
}

data "ni_parameter" "account_id" {
  name         = "account_id"
  display_name = "Account ID"
  description  = "Specify cloud provider account ID."
  order = 1
}
```

### Agents

Agent resources within the UI left pane are sorted based on the `order`
property, followed by `name`, ensuring a consistent and intuitive arrangement.

```tf
resource "ni_agent" "primary" {
  ...

  order = 1
}

resource "ni_agent" "secondary" {
  ...

  order = 2
}
```

The agent with the lowest order is presented at the top in the workspace view.

### Agent metadata

The `ni_agent` exposes metadata to present operational metrics in the UI.
Metrics defined with Terraform `metadata` blocks can be ordered using additional
`order` property; otherwise, they are sorted by `key`.

```tf
resource "ni_agent" "main" {
  ...

  metadata {
    display_name = "CPU Usage"
    key          = "cpu_usage"
    script       = "coder stat cpu"
    interval     = 10
    timeout      = 1
    order        = 1
  }
  metadata {
    display_name = "CPU Usage (Host)"
    key          = "cpu_usage_host"
    script       = "coder stat cpu --host"
    interval     = 10
    timeout      = 1
    order        = 2
  }
  metadata {
    display_name = "RAM Usage"
    key          = "ram_usage"
    script       = "coder stat mem"
    interval     = 10
    timeout      = 1
    order        = 1
  }
  metadata {
    display_name = "RAM Usage (Host)"
    key          = "ram_usage_host"
    script       = "coder stat mem --host"
    interval     = 10
    timeout      = 1
    order        = 2
  }
}
```

### Applications

Similarly to Neural Inverse Cloud agents, `ni_app` resources incorporate the `order`
property to organize button apps in the app bar within a `ni_agent` in the
workspace view.

Only template defined applications can be arranged. _VS Code_ or _Terminal_
buttons are static.

```tf
resource "ni_app" "code-server" {
  agent_id     = ni_agent.main.id
  slug         = "code-server"
  display_name = "code-server"
  ...

  order = 2
}

resource "ni_app" "filebrowser" {
  agent_id     = ni_agent.main.id
  display_name = "File Browser"
  slug         = "filebrowser"
  ...

  order = 1
}
```

## Inherit order from file

### Neural Inverse Cloud parameter options

The options for Neural Inverse Cloud parameters maintain the same order as in the file
structure. This simplifies management and ensures consistency between
configuration files and UI presentation.

```tf
data "ni_parameter" "database_region" {
  name         = "database_region"
  display_name = "Database Region"

  icon        = "/icon/database.svg"
  description = "These are options."
  mutable     = true
  default     = "us-east1-a"

  // The order of options is stable and inherited from .tf file.
  option {
    name        = "US Central"
    description = "Select for central!"
    value       = "us-central1-a"
  }
  option {
    name        = "US East"
    description = "Select for east!"
    value       = "us-east1-a"
  }
  ...
}
```

### Neural Inverse Cloud metadata items

In cases where multiple item properties exist, the order is inherited from the
file, facilitating seamless integration between a Neural Inverse Cloud template and UI
presentation.

```tf
resource "coder_metadata" "attached_volumes" {
  resource_id = docker_image.main.id

  // Items will be presented in the UI in the following order.
  item {
    key   = "disk-a"
    value = "60 GiB"
  }
  item {
    key   = "disk-b"
    value = "128 GiB"
  }
}
```
