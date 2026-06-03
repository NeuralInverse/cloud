# Infrastructure

Learn how to spin up & manage Neural Inverse Cloud infrastructure.

## Architecture

Neural Inverse Cloud is a self-hosted platform that runs on your own servers. For large
deployments, we recommend running the control plane on Kubernetes. Workspaces
can be run as VMs or Kubernetes pods. The control plane (`nicloud`) runs in a
single region. However, workspace proxies, provisioners, and workspaces can run
across regions or even cloud providers for the optimal developer experience.

Learn more about Neural Inverse Cloud's
[architecture, concepts, and dependencies](./architecture.md).

## Reference Architectures

We publish [reference architectures](./validated-architectures/index.md) that
include best practices around Neural Inverse Cloud configuration, infrastructure sizing,
autoscaling, and operational readiness for different deployment sizes (e.g.
`Up to 2000 users`).

## Scale Tests

Use our [scale test utility](./scale-utility.md) that can be run on your Neural Inverse Cloud
deployment to simulate user activity and measure performance.

## Monitoring

See our dedicated [Monitoring](../monitoring/index.md) section for details
around monitoring your Neural Inverse Cloud deployment via a bundled Grafana dashboard, health
check, and/or within your own observability stack via Prometheus metrics.
