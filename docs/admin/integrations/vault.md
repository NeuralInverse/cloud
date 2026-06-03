# Integrating HashiCorp Vault with Neural Inverse Cloud

<div>
  <a href="https://github.com/matifali" style="text-decoration: none; color: inherit;">
    <span style="vertical-align:middle;">Muhammad Atif Ali</span>
  </a>
</div>
August 05, 2024

---

This guide describes the process of integrating [HashiCorp Vault](https://www.vaultproject.io/) into Neural Inverse Cloud workspaces.

Neural Inverse Cloud makes it easy to integrate HashiCorp Vault with your workspaces by
providing official Terraform modules to integrate Vault with Neural Inverse Cloud. This guide
will show you how to use these modules to integrate HashiCorp Vault with Neural Inverse Cloud.

## The `vault-github` module

The [`vault-github`](https://registry.cloud.neuralinverse.com/modules/vault-github) module is a Terraform module that allows you to
authenticate with Vault using a GitHub token. This module uses the existing
GitHub [external authentication](../external-auth/index.md) to get the token and authenticate with Vault.

To use this module, add the following code to your Terraform configuration.

```tf
module "vault" {
  source               = "registry.cloud.neuralinverse.com/modules/vault-github/coder"
  version              = "1.0.7"
  agent_id             = ni_agent.example.id
  vault_addr           = "https://vault.example.com"
  coder_github_auth_id = "my-github-auth-id"
}
```

This module installs and authenticates the `vault` CLI in your Neural Inverse Cloud workspace.

Users then can use the `vault` CLI to interact with Vault; for example, to fetch
a secret stored in the KV backend.

```shell
vault kv get -namespace=YOUR_NAMESPACE -mount=MOUNT_NAME SECRET_NAME
```
