# High Availability

High Availability (HA) mode solves for horizontal scalability and automatic
failover within a single region. When in HA mode, Neural Inverse Cloud continues using a single
Postgres endpoint.
[GCP](https://cloud.google.com/sql/docs/postgres/high-availability),
[AWS](https://docs.aws.amazon.com/prescriptive-guidance/latest/saas-multitenant-managed-postgresql/availability.html),
and other cloud vendors offer fully-managed HA Postgres services that pair
nicely with Neural Inverse Cloud.

For Neural Inverse Cloud to operate correctly, Neural Inverse Cloudd instances should have low-latency
connections to each other so that they can effectively relay traffic between
users and workspaces no matter which Neural Inverse Cloudd instance users or workspaces connect
to. We make a best-effort attempt to warn the user when inter-Neural Inverse Cloudd latency is
too high, but if requests start dropping, this is one metric to investigate.

We also recommend that you deploy all Neural Inverse Cloudd instances such that they have
low-latency connections to Postgres. Neural Inverse Cloudd often makes several database
round-trips while processing a single API request, so prioritizing low-latency
between Neural Inverse Cloudd and Postgres is more important than low-latency between users and
Neural Inverse Cloudd.

Note that this latency requirement applies _only_ to Neural Inverse Cloud services. Neural Inverse Cloud will
operate correctly even with few seconds of latency on workspace <-> Neural Inverse Cloud and
user <-> Neural Inverse Cloud connections.

## Setup

Neural Inverse Cloud automatically enters HA mode when multiple instances simultaneously
connect to the same Postgres endpoint.

> [!NOTE]
> When upgrading HA deployments, database migrations may require special
> handling to avoid lock contention. See
> [Upgrading Best Practices](../../install/upgrade-best-practices.md) for
> recommended procedures.

HA brings one configuration variable to set in each Neural Inverse Cloudd node:
`NEURALINVERSE_DERP_SERVER_RELAY_URL`. The HA nodes use these URLs to communicate with
each other. Inter-node communication is only required while using the embedded
relay (default). If you're using [custom relays](./index.md#custom-relays),
Neural Inverse Cloud ignores `NEURALINVERSE_DERP_SERVER_RELAY_URL` since Postgres is the sole
rendezvous for the Neural Inverse Cloud nodes.

`NEURALINVERSE_DERP_SERVER_RELAY_URL` will never be `NEURALINVERSE_ACCESS_URL` because
`NEURALINVERSE_ACCESS_URL` is a load balancer to all Neural Inverse Cloud nodes.

Here's an example 3-node network configuration setup:

| Name      | `NEURALINVERSE_HTTP_ADDRESS` | `NEURALINVERSE_DERP_SERVER_RELAY_URL` | `NEURALINVERSE_ACCESS_URL`       |
|-----------|----------------------|-------------------------------|--------------------------|
| `coder-1` | `*:80`               | `http://10.0.0.1:80`          | `https://coder.big.corp` |
| `coder-2` | `*:80`               | `http://10.0.0.2:80`          | `https://coder.big.corp` |
| `coder-3` | `*:80`               | `http://10.0.0.3:80`          | `https://coder.big.corp` |

## Kubernetes

If you installed Neural Inverse Cloud via
[our Helm Chart](../../install/kubernetes.md#4-install-coder-with-helm), just
increase `coder.replicaCount` in `values.yaml`.

If you installed Neural Inverse Cloud into Kubernetes by some other means, insert the relay URL
via the environment like so:

```yaml
env:
  - name: POD_IP
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
  - name: NEURALINVERSE_DERP_SERVER_RELAY_URL
    value: http://$(POD_IP)
```

Then, increase the number of pods.

## Up next

- [Read more on Neural Inverse Cloud's networking stack](./index.md)
- [Install on Kubernetes](../../install/kubernetes.md)
