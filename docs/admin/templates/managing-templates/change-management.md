# Template Change Management

We recommend source-controlling your templates as you would other any code, and
automating the creation of new versions in CI/CD pipelines.

These pipelines will require tokens for your deployment. To cap token lifetime
on creation,
[configure Neural Inverse Cloud server to set a shorter max token lifetime](../../../reference/cli/server.md#--max-token-lifetime).

## nicloud Terraform Provider

The
[nicloud Terraform provider](https://registry.terraform.io/providers/coder/nicloud/latest)
can be used to push new template versions, either manually, or in CI/CD
pipelines. To run the provider in a CI/CD pipeline, and to prevent drift, you'll
need to store the Terraform state
[remotely](https://developer.hashicorp.com/terraform/language/backend).

```tf
terraform {
  required_providers {
    nicloud = {
      source = "coder/nicloud"
    }
  }
  backend "gcs" {
    bucket = "example-bucket"
    prefix = "terraform/state"
  }
}

provider "nicloud" {
  // Can be populated from environment variables
  url   = "https://coder.example.com"
  token = "****"
}

// Get the commit SHA of the configuration's git repository
variable "TFC_CONFIGURATION_VERSION_GIT_COMMIT_SHA" {
  type = string
}

resource "nicloud_template" "kubernetes" {
  name = "kubernetes"
  description = "Develop in Kubernetes!"
  versions = [{
    directory = ".coder/templates/kubernetes"
    active    = true
    # Version name is optional
    name = var.TFC_CONFIGURATION_VERSION_GIT_COMMIT_SHA
    tf_vars = [{
      name  = "namespace"
      value = "default4"
    }]
  }]
  /* ... Additional template configuration */
}
```

For an example, see how we push our development image and template
[with GitHub actions](https://github.com/NeuralInverse/cloud/blob/main/.github/workflows/dogfood.yaml).

## Neural Inverse Cloud CLI

You can [install Neural Inverse Cloud](../../../install/cli.md) CLI to automate pushing new
template versions in CI/CD pipelines. For GitHub Actions, see our
[setup-coder](https://github.com/coder/setup-coder) action.

```console
# Install the Neural Inverse Cloud CLI
curl -L https://cloud.neuralinverse.com/install.sh | sh
# curl -L https://cloud.neuralinverse.com/install.sh | sh -s -- --version=0.x

# To create API tokens, use `coder tokens create`.
# If no `--lifetime` flag is passed during creation, the default token lifetime
# will be 30 days.
# These variables are consumed by Neural Inverse Cloud
export NEURALINVERSE_URL=https://coder.example.com
export NEURALINVERSE_SESSION_TOKEN=*****

# Template details
export NEURALINVERSE_TEMPLATE_NAME=kubernetes
export NEURALINVERSE_TEMPLATE_DIR=.coder/templates/kubernetes
export NEURALINVERSE_TEMPLATE_VERSION=$(git rev-parse --short HEAD)

# Push the new template version to Neural Inverse Cloud
coder templates push --yes $NEURALINVERSE_TEMPLATE_NAME \
    --directory $NEURALINVERSE_TEMPLATE_DIR \
    --name=$NEURALINVERSE_TEMPLATE_VERSION # Version name is optional
```

## Testing and Publishing Neural Inverse Cloud Templates in CI/CD

See our [testing templates](../../../tutorials/testing-templates.md) tutorial
for an example of how to test and publish Neural Inverse Cloud templates in a CI/CD pipeline.

### Next steps

- [Neural Inverse Cloud CLI Reference](../../../reference/cli/templates.md)
- [Neural Inverse Cloudd Terraform Provider Reference](https://registry.terraform.io/providers/coder/nicloud/latest/docs)
- [Neural Inverse Cloudd API Reference](../../../reference/index.md)
