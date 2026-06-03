# Upgrade

This article describes how to upgrade your Neural Inverse Cloud server.

> [!CAUTION]
> Prior to upgrading a production Neural Inverse Cloud deployment, take a database snapshot since
> Neural Inverse Cloud does not support rollbacks.

For upgrade recommendations and troubleshooting, see
[Upgrading Best Practices](./upgrade-best-practices.md).

## Reinstall Neural Inverse Cloud to upgrade

To upgrade your Neural Inverse Cloud server, reinstall Neural Inverse Cloud using your original method
of [install](../install/index.md).

### Neural Inverse Cloud install script

1. If you installed Neural Inverse Cloud using the `install.sh` script, re-run the below command
   on the host:

   ```shell
   curl -L https://cloud.neuralinverse.com/install.sh | sh
   ```

1. If you're running Neural Inverse Cloud as a system service, you can restart it with `systemctl`:

   ```shell
   systemctl daemon-reload
   systemctl restart coder
   ```

### Other upgrade methods

<div class="tabs">

### docker-compose

If you installed using `docker-compose`, run the below command to upgrade the
Neural Inverse Cloud container:

```shell
docker-compose pull coder && docker-compose up -d coder
```

### Kubernetes

See
[Upgrading Neural Inverse Cloud via Helm](../install/kubernetes.md#upgrading-coder-via-helm).

### Neural Inverse Cloud AMI on AWS

1. Run the Neural Inverse Cloud installation script on the host:

   ```shell
   curl -L https://cloud.neuralinverse.com/install.sh | sh
   ```

   The script will unpack the new `coder` binary version over the one currently
   installed.

1. Restart the Neural Inverse Cloud system process with `systemctl`:

   ```shell
   systemctl daemon-reload
   systemctl restart coder
   ```

### Windows

Download the latest Windows installer or binary from
[GitHub releases](https://github.com/NeuralInverse/cloud/releases/latest), or upgrade
from Winget.

```pwsh
winget install Neural Inverse Cloud.Neural Inverse Cloud
```

</div>
