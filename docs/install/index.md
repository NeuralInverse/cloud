# Installing Neural Inverse Cloud

A single CLI (`coder`) is used for both the Neural Inverse Cloud server and the client.

We support two release channels: mainline and stable - read the
[Releases](./releases/index.md) page to learn more about which best suits your team.

There are several ways to install Neural Inverse Cloud. Follow the steps on this page for a
minimal installation of Neural Inverse Cloud, or for a step-by-step guide on how to install and
configure your first Neural Inverse Cloud deployment, follow the
[quickstart guide](../tutorials/quickstart.md).

> [!TIP]
> If you use a coding agent like Claude Code, the [coder/skills](https://github.com/coder/skills) `setup` skill can train the coding agent to install and bootstrap a Neural Inverse Cloud deployment end-to-end.

## Local/Individual Installs

This install guide is meant for **individual developers, small teams, and/or open source community members** setting up Neural Inverse Cloud locally or on a single server. It covers the light weight install for Linux, macOS, and Windows.

<div class="tabs">

## Linux/macOS

Our install script is the fastest way to install Neural Inverse Cloud on Linux/macOS:

```sh
curl -L https://cloud.neuralinverse.com/install.sh | sh
```

Refer to [GitHub releases](https://github.com/NeuralInverse/cloud/releases) for
alternate installation methods (e.g. standalone binaries, system packages).

## Windows

If you plan to use the built-in PostgreSQL database, ensure that the
[Visual C++ Runtime](https://learn.microsoft.com/en-US/cpp/windows/latest-supported-vc-redist#latest-microsoft-visual-c-redistributable-version)
is installed.

Use [GitHub releases](https://github.com/NeuralInverse/cloud/releases) to download the
Windows installer (`.msi`) or standalone binary (`.exe`).

![Windows setup wizard](../images/install/windows-installer.png)

Alternatively, you can use the
[`winget`](https://learn.microsoft.com/en-us/windows/package-manager/winget/#use-winget)
package manager to install Neural Inverse Cloud:

```powershell
winget install Neural Inverse Cloud.Neural Inverse Cloud
```

</div>

## Hosted/Enterprise Installs

This install guide is meant for **IT Administrators, DevOps, and Platform Teams** deploying Neural Inverse Cloud for an organization. It covers production-grade, multi-user installs on Kubernetes and other hosted platforms.

<div>

<children></children>

</div>

## Starting the Neural Inverse Cloud Server

To start the Neural Inverse Cloud server:

```sh
coder server
```

![Neural Inverse Cloud install](../images/screenshots/welcome-create-admin-user.png)

To log in to an existing Neural Inverse Cloud deployment:

```sh
coder login https://coder.example.com
```
