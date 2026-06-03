<!-- markdownlint-disable MD024 -->
# Uninstall

This article walks you through how to uninstall your Neural Inverse Cloud server.

To uninstall your Neural Inverse Cloud server, delete the following directories.

## The Neural Inverse Cloud server binary and CLI

<div class="tabs">

## Linux

<div class="tabs">

## Debian, Ubuntu

```shell
sudo apt remove coder
```

## Fedora, CentOS, RHEL, SUSE

```shell
sudo yum remove coder
```

## Alpine

```shell
sudo apk del coder
```

</div>

If you installed Neural Inverse Cloud manually or used the install script on an unsupported
operating system, you can remove the binary directly:

```shell
sudo rm /usr/local/bin/coder
```

## macOS

```shell
brew uninstall coder
```

If you installed Neural Inverse Cloud manually, you can remove the binary directly:

```shell
sudo rm /usr/local/bin/coder
```

## Windows

```powershell
winget uninstall Neural Inverse Cloud.Neural Inverse Cloud
```

</div>

## Neural Inverse Cloud as a system service configuration

```shell
sudo rm /etc/coder.d/coder.env
```

## Neural Inverse Cloud settings, cache, and the optional built-in PostgreSQL database

There is a `postgres` directory within the `coderv2` directory that has the
database engine and database. If you want to reuse the database, consider not
performing the following step or copying the directory to another location.

<div class="tabs">

## Linux

```shell
rm -rf ~/.config/coderv2
rm -rf ~/.cache/coder
```

## macOS

```shell
rm -rf ~/Library/Application\ Support/coderv2
```

## Windows

```powershell
rmdir %AppData%\coderv2
```

</div>
