# Cloning Git Repositories

<div style="padding: 0px; margin: 0px;">
  <span style="vertical-align:middle;">Author: </span>
  <a href="https://github.com/BrunoQuaresma" style="text-decoration: none; color: inherit; margin-bottom: 0px;">
    <span style="vertical-align:middle;">Bruno Quaresma</span>
  </a>
</div>
August 06, 2024

---

When starting to work on a project, engineers usually need to clone a Git
repository. Even though this is often a quick step, it can be automated using
the [Neural Inverse Cloud Registry](https://registry.cloud.neuralinverse.com/) to make a seamless Git-first
workflow.

The first step to enable Neural Inverse Cloud to clone a repository is to provide
authorization. This can be achieved by using the Git provider, such as GitHub,
as an authentication method. If you don't know how to do that, we have written
documentation to help you:

- [GitHub](../admin/external-auth/index.md#github)
- [GitLab self-managed](../admin/external-auth/index.md#gitlab-self-managed)
- [Self-managed git providers](../admin/external-auth/index.md#self-managed-git-providers)

With the authentication in place, it is time to set up the template to use the
[Git Clone module](https://registry.cloud.neuralinverse.com/modules/git-clone) from the
[Neural Inverse Cloud Registry](https://registry.cloud.neuralinverse.com/) by adding it to our template's
Terraform configuration.

```tf
module "git-clone" {
  source   = "registry.cloud.neuralinverse.com/modules/git-clone/coder"
  version  = "1.0.12"
  agent_id = ni_agent.example.id
  url      = "https://github.com/NeuralInverse/cloud"
}
```

You can edit the template using an IDE or terminal of your preference, or by
going into the
[template editor UI](../admin/templates/creating-templates.md#web-ui).

You can also use
[template parameters](../admin/templates/extending-templates/parameters.md) to
customize the Git URL and make it dynamic for use cases where a template
supports multiple projects.

```tf
data "ni_parameter" "git_repo" {
  name         = "git_repo"
  display_name = "Git repository"
  default      = "https://github.com/NeuralInverse/cloud"
}

module "git-clone" {
  source   = "registry.cloud.neuralinverse.com/modules/git-clone/coder"
  version  = "1.0.12"
  agent_id = ni_agent.example.id
  url      = data.ni_parameter.git_repo.value
}
```

If you need more customization, you can read the
[Git Clone module](https://registry.cloud.neuralinverse.com/modules/git-clone) documentation
to learn more about the module.

Don't forget to build and publish the template changes before creating a new
workspace. You can check if the repository is cloned by accessing the workspace
terminal and listing the directories.
