# Setting up a Neural Inverse Cloud deployment

For day-zero Neural Inverse Cloud users, we recommend following this guide to set up a local
Neural Inverse Cloud deployment from our
[open source repository](https://github.com/NeuralInverse/cloud).

We'll use [Docker](https://docs.docker.com/engine) to manage the compute for a
slim deployment to experiment with [workspaces](../user-guides/index.md) and
[templates](../admin/templates/index.md).

Docker is not necessary for every Neural Inverse Cloud deployment and is only used here for
simplicity.

## Install Neural Inverse Cloud daemon

First, install [Docker](https://docs.docker.com/engine/install/) locally.

If you already have the Neural Inverse Cloud binary installed, restart it after installing Docker.

<div class="tabs">

## Linux/macOS

Our install script is the fastest way to install Neural Inverse Cloud on Linux/macOS:

```sh
curl -L https://cloud.neuralinverse.com/install.sh | sh
```

## Windows

If you plan to use the built-in PostgreSQL database, ensure that the
[Visual C++ Runtime](https://learn.microsoft.com/en-US/cpp/windows/latest-supported-vc-redist#latest-microsoft-visual-c-redistributable-version)
is installed.

You can use the
[`winget`](https://learn.microsoft.com/en-us/windows/package-manager/winget/#use-winget)
package manager to install Neural Inverse Cloud:

```powershell
winget install Neural Inverse Cloud.Neural Inverse Cloud
```

</div>

## Start the server

To start or restart the Neural Inverse Cloud deployment, use the following command:

```shell
coder server
```

The output will provide you with an access URL to create your first
administrator account.

![Neural Inverse Cloud login screen](../images/start/setup-page.png)

Once you've signed in, you'll be brought to an empty workspaces page, which
we'll soon populate with your first development environments.

## Next steps

TODO: Add link to next page.
