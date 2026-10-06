# domain-expiry-exporter

Prometheus exporter for domain expiry monitoring. Domains are checked via RDAP
(with automatic WHOIS fallback) by a background scheduler, so scrapes never wait
for registrars. Designed for fleets of hundreds to thousands of domains.

`/metrics` and `/probe` only read from an in-memory registry and never make a
network call, so scrape latency and registrar availability are independent of
each other.

## Features

- RDAP lookups with automatic WHOIS fallback
- Adaptive check intervals: domains close to expiry are checked more often
- Exponential backoff on repeated failures, exposed as a metric
- Bounded concurrency for registrar lookups
- Blackbox-style `/probe` endpoint: the first request registers a domain, no
  domain list is needed
- Automatic eviction of domains that are no longer scraped
- Optional static domain list and optional Redis persistence

## How it works

1. Prometheus requests `/probe?target=example.com`. The exporter records the
   request and answers from its registry.
2. A background scheduler checks each known domain when its next check is due:
   hourly if it expires in under 7 days, every 12 hours if under 30 days, daily
   otherwise.
3. A failed check is retried with a growing delay (1 minute, doubling up to
   1 hour) and counted in `domain_consecutive_failures`.
4. A domain that Prometheus has not requested for 24 hours is forgotten.

Until the first successful check of a new domain, only
`domain_consecutive_failures` is exposed for it.

## Quick start

```bash
docker run -d -p 9222:9222 ghcr.io/dr1m91/domain-expiry-exporter:<version>
```

Prometheus scrape config (blackbox-style, domains are discovered from targets):

```yaml
- job_name: domain-expiry
  metrics_path: /probe
  static_configs:
    - targets: ["example.com", "another-example.com"]
  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_target
    - target_label: __address__
      replacement: domain-expiry-exporter:9222
```

Or seed a static list with `--config=domains.yaml`. These domains are never
forgotten:

```yaml
domains:
  - example.com
  - another-example.com
```

Run with `--help` for all flags. The ones you are most likely to change are
`--scan-interval` (default `1m`), `--concurrency` (default `20`) and
`--stale-after` (default `24h`).

## Persistence

By default the cache lives in memory and is rebuilt after a restart. With
`--redis-addr` the exporter also keeps a copy in Redis, so a restart or an
update does not re-check every domain at once.

Memory stays the source of truth: `/probe` and `/metrics` never wait for Redis.
Changes are flushed in the background every 10 seconds and once more on
shutdown. If Redis is down, the exporter keeps working and retries the writes.
State is per instance: run a single replica per Redis.

## Helm chart

```bash
helm install domain-expiry-exporter oci://ghcr.io/dr1m91/charts/domain-expiry-exporter --version <version>
```

To keep the cache in Redis (or Valkey):

```yaml
redis:
  enabled: true
  host: valkey-redis
  username: default
  passwordSecret:
    name: redis
    key: password
```

Or let the chart run Valkey next to the exporter. Create a secret with a
`password` key first:

```bash
kubectl create secret generic redis --from-literal=password="$(openssl rand -hex 16)"
```

```yaml
valkey:
  enabled: true
  auth:
    usersExistingSecret: redis
```

See [values.yaml](./charts/domain-expiry-exporter/values.yaml) for all options.

## Metrics

| Metric | Description |
|---|---|
| `domain_expiry_days` | Days until the domain expires |
| `domain_probe_success` | Whether the domain has ever been successfully probed |
| `domain_last_success_timestamp_seconds` | Unix timestamp of the last successful probe |
| `domain_consecutive_failures` | Number of consecutive failed probe attempts since the last success |

All metrics carry a `domain` label.

## Origin

Started from [caarlos0/domain_exporter](https://github.com/caarlos0/domain_exporter)
(archived August 2026) and reuses its RDAP and WHOIS client code. Many thanks
to Carlos Alexandro Becker for the original work.

The architecture is different: lookups run in a background scheduler with
adaptive intervals instead of being triggered by scrapes, domains are tracked
in a registry fed by scrape requests, and the state can be persisted in Redis.

## License

MIT, see [LICENSE](./LICENSE). Portions derived from
[caarlos0/domain_exporter](https://github.com/caarlos0/domain_exporter) (MIT, 2017).
