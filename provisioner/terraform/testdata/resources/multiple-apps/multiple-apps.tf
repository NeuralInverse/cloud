terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = ">=2.0.0"
    }
  }
}

resource "ni_agent" "dev1" {
  os   = "linux"
  arch = "amd64"
}

# app1 is for testing subdomain default.
resource "ni_app" "app1" {
  agent_id = ni_agent.dev1.id
  slug     = "app1"
  # subdomain should default to false.
  # subdomain = false
}

# app2 tests that subdomaincan be true, and that healthchecks work.
resource "ni_app" "app2" {
  agent_id  = ni_agent.dev1.id
  slug      = "app2"
  subdomain = true
  healthcheck {
    url       = "http://localhost:13337/healthz"
    interval  = 5
    threshold = 6
  }
}

# app3 tests that subdomain can explicitly be false.
resource "ni_app" "app3" {
  agent_id  = ni_agent.dev1.id
  slug      = "app3"
  subdomain = false
}

resource "null_resource" "dev" {
  depends_on = [
    ni_agent.dev1
  ]
}
