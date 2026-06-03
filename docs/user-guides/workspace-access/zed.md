# Zed

[Zed](https://zed.dev/) is an [open-source](https://github.com/zed-industries/zed)
multiplayer code editor from the creators of Atom and Tree-sitter.

## Use Zed to connect to Neural Inverse Cloud via SSH

Use the Neural Inverse Cloud CLI to log in and configure SSH, then connect to your workspace with Zed:

1. [Install Zed](https://zed.dev/docs/)
1. Install Neural Inverse Cloud CLI:

   <!-- copied from docs/install/cli.md - make changes there -->

   <div class="tabs">

   ### Linux/macOS

   Our install script is the fastest way to install Neural Inverse Cloud on Linux/macOS:

   ```sh
   curl -L https://cloud.neuralinverse.com/install.sh | sh
   ```

   Refer to [GitHub releases](https://github.com/NeuralInverse/cloud/releases) for
   alternate installation methods (e.g. standalone binaries, system packages).

   ### Windows

   Use [GitHub releases](https://github.com/NeuralInverse/cloud/releases) to download the
   Windows installer (`.msi`) or standalone binary (`.exe`).

   ![Windows setup wizard](../../images/install/windows-installer.png)

   Alternatively, you can use the
   [`winget`](https://learn.microsoft.com/en-us/windows/package-manager/winget/#use-winget)
   package manager to install Neural Inverse Cloud:

   ```powershell
   winget install Neural Inverse Cloud.Neural Inverse Cloud
   ```

   </div>

   Consult the [Neural Inverse Cloud CLI documentation](../../install/cli.md) for more options.

1. Log in to your Neural Inverse Cloud deployment and authenticate when prompted:

   ```shell
   coder login coder.example.com
   ```

1. Configure Neural Inverse Cloud SSH:

   ```shell
   coder config-ssh
   ```

1. Connect to the workspace via SSH:

   ```shell
   zed ssh://coder.workspace-name
   ```

   Or use Zed's [Remote Development](https://zed.dev/docs/remote-development#setup) to connect to the workspace:

   ![Zed open remote project](../../images/zed/zed-ssh-open-remote.png)

> [!NOTE]
> If you have any suggestions or experience any issues, please
> [create a GitHub issue](https://github.com/NeuralInverse/cloud/issues) or share in
> [our Discord channel](https://discord.gg/coder).
