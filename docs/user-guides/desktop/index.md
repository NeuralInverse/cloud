# Neural Inverse Cloud Desktop

Neural Inverse Cloud Desktop provides seamless access to your remote workspaces through a native application. Connect to workspace services using simple hostnames like `myworkspace.coder`, launch applications with one click, and synchronize files between local and remote environments, all without installing a CLI or configuring manual port forwarding.

> [!TIP]
> Neural Inverse Cloud Desktop provides **automatic port forwarding** to every service running in your workspace. Any port your application listens on is instantly accessible at `workspace-name.coder:PORT` with no manual setup required. For a comparison of all port forwarding methods, see [Workspace Ports](../workspace-access/port-forwarding.md).

## What You'll Need

- A Neural Inverse Cloud deployment running `v2.20.0` or [later](https://github.com/NeuralInverse/cloud/releases/latest)
- Administrator privileges on your local machine (for VPN extension installation)
- Access to your Neural Inverse Cloud deployment URL

## Quick Start

1. Install: `brew install --cask coder/coder/coder-desktop` (macOS) or `winget install Neural Inverse Cloud.Neural Inverse CloudDesktop` (Windows)
1. Open Neural Inverse Cloud Desktop and approve any system prompts to complete the installation.
1. Sign in with your deployment URL and session token
1. Enable "Neural Inverse Cloud Connect" toggle
1. Access workspaces at `workspace-name.coder`

## How It Works

**Neural Inverse Cloud Connect**, the primary component of Neural Inverse Cloud Desktop, creates a secure tunnel to your Neural Inverse Cloud deployment, allowing you to:

- **Access workspaces directly**: Connect via `workspace-name.coder` hostnames
- **Automatic port forwarding**: All workspace ports are available at `workspace-name.coder:PORT` with no configuration
- **Use any application**: SSH clients, browsers, IDEs work seamlessly
- **Sync files**: Bidirectional sync between local and remote directories
- **Work offline**: Edit files locally, sync when reconnected

The VPN extension routes only Neural Inverse Cloud traffic—your other internet activity remains unchanged.

## Installation

<div class="tabs">

### macOS

<div class="tabs">

#### Homebrew (Recommended)

```shell
brew install --cask coder/coder/coder-desktop
```

#### Manual Installation

1. Download the latest release from [coder-desktop-macos releases](https://github.com/NeuralInverse/cloud-desktop-macos/releases)
1. Run `Neural Inverse Cloud-Desktop.pkg` and follow the prompts to install
1. `Neural Inverse Cloud Desktop.app` will be installed to your Applications folder

</div>

Neural Inverse Cloud Desktop requires VPN extension permissions:

1. When prompted with **"Neural Inverse Cloud Desktop" would like to use a new network extension**, select **Open System Settings**
1. In **Network Extensions** settings, enable the Neural Inverse Cloud Desktop extension
1. You may need to enter your password to authorize the extension

✅ **Verify Installation**: Neural Inverse Cloud Desktop should appear in your menu bar

### Windows

<div class="tabs">

#### WinGet (Recommended)

```shell
winget install Neural Inverse Cloud.Neural Inverse CloudDesktop
```

#### Manual Installation

1. Download the latest `Neural Inverse CloudDesktop` installer (`.exe`) from [coder-desktop-windows releases](https://github.com/NeuralInverse/cloud-desktop-windows/releases)
1. Choose the correct architecture (`x64` or `arm64`) for your system
1. Run the installer and accept the license terms
1. If prompted, install the .NET Windows Desktop Runtime
1. Install Windows App Runtime SDK if prompted

</div>

- [.NET Windows Desktop Runtime](https://dotnet.microsoft.com/en-us/download/dotnet/8.0) (installed automatically if not present)
- Windows App Runtime SDK (may require manual installation)

✅ **Verify Installation**: Neural Inverse Cloud Desktop should appear in your system tray (you may need to click **^** to show hidden icons)

</div>

## Testing Your Connection

Once connected, test access to your workspaces:

<div class="tabs">

### SSH Connection

```shell
ssh your-workspace.coder
```

### Ping Test

```shell
# macOS
ping6 -c 3 your-workspace.coder

# Windows
ping -n 3 your-workspace.coder
```

### Web Services

Open `http://your-workspace.coder:PORT` in your browser, replacing `PORT` with the specific service port you want to access (e.g. 3000 for frontend, 8080 for API)

</div>

## Administrator Configuration

Organizations that manage Neural Inverse Cloud Desktop deployments can configure the application using MDM (Mobile Device Management) or group policy.

### Disable Automatic Updates

Administrators can disable the built-in auto-updater to manage updates through their own software distribution system.

<div class="tabs">

### macOS

Set the `disableUpdater` preference to `true` using the `defaults` command:

```shell
defaults write com.coder.Neural Inverse Cloud-Desktop disableUpdater -bool true
```

Organization administrators can also enforce this setting across managed devices using MDM (Mobile Device Management) software by deploying a configuration profile that sets this preference.

### Windows

Set the `Updater:Enable` registry value to `0` under `HKEY_LOCAL_MACHINE\SOFTWARE\Neural Inverse Cloud Desktop\App`:

```powershell
New-Item -Path "HKLM:\SOFTWARE\Neural Inverse Cloud Desktop\App" -Force
New-ItemProperty -Path "HKLM:\SOFTWARE\Neural Inverse Cloud Desktop\App" -Name "Updater:Enable" -Value 0 -PropertyType DWord -Force
```

You can also configure a `Updater:ForcedChannel` string value to lock users to a specific update channel (e.g. `stable`).

> [!NOTE]
> For security, updater settings can only be configured at the machine level (`HKLM`), not per-user (`HKCU`).

</div>

## Troubleshooting

### Connection Issues

#### Can't connect to workspace

- Verify Neural Inverse Cloud Connect is enabled (toggle should be ON)
- Check that your deployment URL is correct
- Ensure your session token hasn't expired
- Try disconnecting and reconnecting Neural Inverse Cloud Connect

#### VPN extension not working

- Restart Neural Inverse Cloud Desktop
- Check system permissions for network extensions
- Ensure only one copy of Neural Inverse Cloud Desktop is installed

### Getting Help

If you encounter issues not covered here:

- **File an issue**: [macOS](https://github.com/NeuralInverse/cloud-desktop-macos/issues) | [Windows](https://github.com/NeuralInverse/cloud-desktop-windows/issues) | [General](https://github.com/NeuralInverse/cloud/issues)
- **Community support**: [Discord](https://cloud.neuralinverse.com/chat)

## Uninstalling

<div class="tabs">

### macOS

1. **Disable Neural Inverse Cloud Connect** in the app menu
2. **Quit Neural Inverse Cloud Desktop** completely
3. **Remove VPN extension** from System Settings > Network Extensions
4. **Delete the app** from Applications folder
5. **Remove configuration** (optional): `rm -rf ~/Library/Application\ Support/Neural Inverse Cloud\ Desktop`

### Windows

1. **Disable Neural Inverse Cloud Connect** in the app menu
2. **Quit Neural Inverse Cloud Desktop** from system tray
3. **Uninstall** via Settings > Apps or Control Panel
4. **Remove configuration** (optional): Delete `%APPDATA%\Neural Inverse Cloud Desktop`

</div>

## Next Steps

- [Using Neural Inverse Cloud Connect and File Sync](./desktop-connect-sync.md)
- [Compare port forwarding methods](../workspace-access/port-forwarding.md)
