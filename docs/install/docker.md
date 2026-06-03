# Install Neural Inverse Cloud via Docker

You can install and run Neural Inverse Cloud using the official Docker images published on
[GitHub Container Registry](https://github.com/NeuralInverse/cloud/pkgs/container/coder).

## Requirements

- Docker. See the
  [official installation documentation](https://docs.docker.com/install/).

- A Linux host.

- 2 CPU cores and 4 GB memory free on your machine.

> [!IMPORTANT]
> This guide is for **Linux** hosts only. The `getent` and `--group-add`
> Docker socket patterns used below are Linux-specific and do not translate
> cleanly to macOS Docker runtimes. For macOS, install Neural Inverse Cloud using the
> [standalone binary](./cli.md) instead.

<div class="tabs">

## Install Neural Inverse Cloud via `docker compose`

Neural Inverse Cloud publishes a
[docker compose example](https://github.com/NeuralInverse/cloud/blob/main/compose.yaml)
which includes a PostgreSQL container and volume.

1. Make sure you have [Docker Compose](https://docs.docker.com/compose/install/)
   installed.

1. Download the
   [`docker-compose.yaml`](https://github.com/NeuralInverse/cloud/blob/main/compose.yaml)
   file.

1. Update `group_add:` in `docker-compose.yaml` with the `gid` of `docker`
   group. You can get the `docker` group `gid` by running the below command:

   ```shell
   getent group docker | cut -d: -f3
   ```

1. Start Neural Inverse Cloud with `docker compose up`

1. Visit the web UI via the configured url.

1. Follow the on-screen instructions log in and create your first template and
   workspace

Neural Inverse Cloud configuration is defined via environment variables. Learn more about
Neural Inverse Cloud's [configuration options](../admin/setup/index.md).

## Install Neural Inverse Cloud via `docker run`

### Built-in database (quick)

For proof-of-concept deployments, you can run a complete Neural Inverse Cloud instance with the
following command.

```shell
export NEURALINVERSE_DATA=$HOME/.config/coderv2-docker
export DOCKER_GROUP=$(getent group docker | cut -d: -f3)
mkdir -p $NEURALINVERSE_DATA
docker run --rm -it \
  -v $NEURALINVERSE_DATA:/home/coder/.config \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --group-add $DOCKER_GROUP \
  ghcr.io/coder/coder:latest
```

### External database (recommended)

For production deployments, we recommend using an external PostgreSQL database
(version 13 or higher). Set `NEURALINVERSE_ACCESS_URL` to the external URL that users
and workspaces will use to connect to Neural Inverse Cloud.

```shell
export DOCKER_GROUP=$(getent group docker | cut -d: -f3)
docker run --rm -it \
  -e NEURALINVERSE_ACCESS_URL="https://coder.example.com" \
  -e NEURALINVERSE_PG_CONNECTION_URL="postgresql://username:password@database/coder" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --group-add $DOCKER_GROUP \
  ghcr.io/coder/coder:latest
```

</div>

## Install the preview release

> [!TIP]
> We do not recommend using preview releases in production environments.

You can install and test a
[preview release of Neural Inverse Cloud](https://github.com/NeuralInverse/cloud/pkgs/container/coder-preview)
by using the `coder-preview:latest` image tag.
This image is automatically updated with the latest changes from the `main` branch.

Replace `ghcr.io/coder/coder:latest` in the `docker run` command in the
[steps above](#install-coder-via-docker-run) with `ghcr.io/coder/coder-preview:latest`.

## Troubleshooting

### Cannot connect to the Docker daemon

If you see an error like:

```text
Error: Error pinging Docker server: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
```

Docker is not installed or not running on the host. Install Docker and start the
daemon before creating a workspace from a Docker-based template. Refer to the
[quickstart troubleshooting](../tutorials/quickstart.md#cannot-connect-to-the-docker-daemon)
for platform-specific steps.

### Docker-based workspace is stuck in "Connecting..."

Ensure you have an externally-reachable `NEURALINVERSE_ACCESS_URL` set. See
[troubleshooting templates](../admin/templates/troubleshooting.md) for more
steps.

### Permission denied while trying to connect to the Docker daemon socket

See Docker's official documentation to
[Manage Docker as a non-root user](https://docs.docker.com/engine/install/linux-postinstall/#manage-docker-as-a-non-root-user)

### I cannot add Docker templates

Neural Inverse Cloud runs as a non-root user, we use `--group-add` to ensure Neural Inverse Cloud has
permissions to manage Docker via `docker.sock`. If the host systems
`/var/run/docker.sock` is not group writable or does not belong to the `docker`
group, the above may not work as-is.

### I cannot add cloud-based templates

In order to use cloud-based templates (e.g. Kubernetes, AWS), you must have an
external URL that users and workspaces will use to connect to Neural Inverse Cloud. For
proof-of-concept deployments, you can use
[Neural Inverse Cloud's tunnel](../admin/setup/index.md#tunnel). For production deployments, we
recommend setting an [access URL](../admin/setup/index.md#access-url)

## Next steps

- [Create your first template](../tutorials/template-from-scratch.md)
- [Control plane configuration](../admin/setup/index.md#configure-control-plane-access)
