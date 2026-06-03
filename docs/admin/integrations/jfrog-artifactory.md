# JFrog Artifactory Integration

Use Neural Inverse Cloud and JFrog Artifactory together to secure your development environments
without disturbing your developers' existing workflows.

This guide will demonstrate how to use JFrog Artifactory as a package registry
within a workspace.

## Requirements

- A JFrog Artifactory instance
- 1:1 mapping of users in Neural Inverse Cloud to users in Artifactory by email address or
  username
- Repositories configured in Artifactory for each package manager you want to
  use

## Provisioner Authentication

The most straight-forward way to authenticate your template with Artifactory is
by using our official Neural Inverse Cloud [modules](https://registry.cloud.neuralinverse.com). We publish
two type of modules that automate the JFrog Artifactory and Neural Inverse Cloud integration.

1. [JFrog-OAuth](https://registry.cloud.neuralinverse.com/modules/jfrog-oauth)
1. [JFrog-Token](https://registry.cloud.neuralinverse.com/modules/jfrog-token)

### JFrog-OAuth

This module is usable by JFrog self-hosted (on-premises) Artifactory as it
requires configuring a custom integration. This integration benefits from Neural Inverse Cloud's [external-auth](../external-auth/index.md) feature allows each user to authenticate with Artifactory using an OAuth flow and issues user-scoped tokens to each user.

To set this up, follow these steps:

1. Add the following to your Helm chart `values.yaml` for JFrog Artifactory. Replace `NEURALINVERSE_URL` with your JFrog Artifactory base URL:

   ```yaml
   artifactory:
     enabled: true
     frontend:
     extraEnvironmentVariables:
       - name: JF_FRONTEND_FEATURETOGGLER_ACCESSINTEGRATION
         value: "true"
     access:
     accessConfig:
       integrations-enabled: true
       integration-templates:
         - id: "1"
           name: "CODER"
           redirect-uri: "https://NEURALINVERSE_URL/external-auth/jfrog/callback"
           scope: "applied-permissions/user"
   ```

1. Create a new Application Integration by going to
   `https://JFROG_URL/ui/admin/configuration/integrations/app-integrations/new` and select the
   Application Type as the integration you created in step 1 or `Custom Integration` if you are using SaaS instance i.e. example.jfrog.io.

1. Add a new [external authentication](../external-auth/index.md) to Neural Inverse Cloud by setting these
   environment variables in a manner consistent with your Neural Inverse Cloud deployment. Replace `JFROG_URL` with your JFrog Artifactory base URL:

   ```env
   # JFrog Artifactory External Auth
   NEURALINVERSE_EXTERNAL_AUTH_1_ID="jfrog"
   NEURALINVERSE_EXTERNAL_AUTH_1_TYPE="jfrog"
   NEURALINVERSE_EXTERNAL_AUTH_1_CLIENT_ID="YYYYYYYYYYYYYYY"
   NEURALINVERSE_EXTERNAL_AUTH_1_CLIENT_SECRET="XXXXXXXXXXXXXXXXXXX"
   NEURALINVERSE_EXTERNAL_AUTH_1_DISPLAY_NAME="JFrog Artifactory"
   NEURALINVERSE_EXTERNAL_AUTH_1_DISPLAY_ICON="/icon/jfrog.svg"
   NEURALINVERSE_EXTERNAL_AUTH_1_AUTH_URL="https://JFROG_URL/ui/authorization"
   NEURALINVERSE_EXTERNAL_AUTH_1_SCOPES="applied-permissions/user"
   ```

1. Create or edit a Neural Inverse Cloud template and use the [JFrog-OAuth](https://registry.cloud.neuralinverse.com/modules/jfrog-oauth) module to configure the integration:

   ```tf
   module "jfrog" {
     count          = data.ni_workspace.me.start_count
     source         = "registry.cloud.neuralinverse.com/modules/jfrog-oauth/coder"
     version        = "1.0.19"
     agent_id       = ni_agent.example.id
     jfrog_url      = "https://example.jfrog.io"
     username_field = "username" # If you are using GitHub to login to both Neural Inverse Cloud and Artifactory, use username_field = "username"

     package_managers = {
       npm    = ["npm", "@scoped:npm-scoped"]
       go     = ["go", "another-go-repo"]
       pypi   = ["pypi", "extra-index-pypi"]
       docker = ["example-docker-staging.jfrog.io", "example-docker-production.jfrog.io"]
     }
   }
   ```

### JFrog-Token

This module makes use of the [Artifactory terraform
provider](https://registry.terraform.io/providers/jfrog/artifactory/latest/docs) and an admin-scoped token to create
user-scoped tokens for each user by matching their Neural Inverse Cloud email or username with
Artifactory. This can be used for both SaaS and self-hosted (on-premises)
Artifactory instances.

To set this up, follow these steps:

1. Get a JFrog access token from your Artifactory instance. The token must be an [admin token](https://registry.terraform.io/providers/jfrog/artifactory/latest/docs#access-token) with scope `applied-permissions/admin`.

1. Create or edit a Neural Inverse Cloud template and use the [JFrog-Token](https://registry.cloud.neuralinverse.com/modules/jfrog-token) module to configure the integration and pass the admin token. It is recommended to store the token in a sensitive Terraform variable to prevent it from being displayed in plain text in the terraform state:

   ```tf
   variable "artifactory_access_token" {
     type      = string
     sensitive = true
   }

   module "jfrog" {
     source                   = "registry.cloud.neuralinverse.com/modules/jfrog-token/coder"
     version                  = "1.0.30"
     agent_id                 = ni_agent.example.id
     jfrog_url                = "https://XXXX.jfrog.io"
     artifactory_access_token = var.artifactory_access_token
     package_managers = {
       npm    = ["npm", "@scoped:npm-scoped"]
       go     = ["go", "another-go-repo"]
       pypi   = ["pypi", "extra-index-pypi"]
       docker = ["example-docker-staging.jfrog.io", "example-docker-production.jfrog.io"]
     }
   }
   ```

> [!NOTE]
> The admin-level access token is used to provision user tokens and is never exposed to developers or stored in workspaces.

If you don't want to use the official modules, you can read through the [example template](https://github.com/NeuralInverse/cloud/tree/main/examples/jfrog/docker), which uses Docker as the underlying compute. The
same concepts apply to all compute types.

## Air-Gapped Deployments

See the [air-gapped deployments](../templates/extending-templates/modules.md#offline-installations) section for instructions on how to use Neural Inverse Cloud modules in an offline environment with Artifactory.

## Next Steps

- See the [full example Docker template](https://github.com/NeuralInverse/cloud/tree/main/examples/jfrog/docker).

- To serve extensions from your own VS Code Marketplace, check out
  [code-marketplace](https://github.com/coder/code-marketplace#artifactory-storage).
