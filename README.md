# domain-expiry-exporter

Prometheus exporter for domain expiry monitoring. Domains are checked via RDAP
(with automatic WHOIS fallback) by a background scheduler, so scrapes never wait
for registrars. Designed for fleets of hundreds to thousands of domains.

`/metrics` and `/probe` only read from an in-memory registry and never make a
network call, so scrape latency and registrar availability are independent of
each other.

## Features

- RDAP lookups with automatic WHOIS fallback
- Adaptive check intervals: domains close to expiry are checked more often,
  domains far from expiry are checked less often
- Exponential backoff on repeated failures, exposed as a metric
- Bounded concurrency for registrar lookups
- Blackbox-style `/probe` endpoint: the first request registers a domain, no
  domain list is needed
- Automatic eviction of domains that are no longer scraped
- Optional static domain list via `--config`
- Optional Redis persistence, so the cache survives restarts

## How it works

1. Prometheus requests `/probe?target=example.com`. The exporter records the
   request and answers from its registry.
2. A background scheduler checks each known domain when its next check is due:
   hourly if it expires in under 7 days, every 12 hours if under 30 days, daily
   otherwise.
3. A failed check is retried with a growing delay (1 minute, doubling up to
   1 hour) and counted in `domain_consecutive_failures`.
4. A domain that Prometheus has not requested for `--stale-after` is forgotten.

Until the first successful check of a new domain, only
`domain_consecutive_failures` is exposed for it.

## Usage

```bash
domain-expiry-exporter \
  --bind=:9222 \
  --scan-interval=1m \
  --stale-after=24h \
  --concurrency=20
```

With Docker:

```bash
docker run -d -p 9222:9222 ghcr.io/dr1m91/domain-expiry-exporter:<version>
```

Prometheus scrape config (blackbox-style, dynamic domain discovery):

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

Or seed a static list at startup with `--config=domains.yaml`. These domains
are never forgotten:

```yaml
domains:
  - example.com
  - another-example.com
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--bind` | `:9222` | Address to listen on |
| `--scan-interval` | `1m` | How often the scheduler looks for domains that are due for a check |
| `--concurrency` | `20` | Maximum number of simultaneous RDAP/WHOIS checks |
| `--check-timeout` | `30s` | Timeout of a single check |
| `--stale-after` | `24h` | Forget a domain that Prometheus has not requested for this long |
| `--config` | | Optional YAML list of domains that are never forgotten |
| `--redis-addr` | | Redis address (`host:port`); empty keeps the cache in memory only |
| `--redis-username` | | Redis ACL username |
| `--redis-password` | | Redis password, also read from the `REDIS_PASSWORD` variable |
| `--redis-key` | `domain-expiry-exporter:entries` | Redis hash that holds the persisted entries |
| `--persist-interval` | `10s` | How often changes are flushed to Redis |
| `--debug` | `false` | Verbose logs |
| `--logFormat` | `console` | `console` or `json` |

## Persistence

By default the cache lives in memory and is rebuilt after a restart. With
`--redis-addr` the exporter also keeps a copy in Redis, so a restart or an
update does not re-check every domain at once.

Memory stays the source of truth: `/probe` and `/metrics` never wait for Redis.
The exporter loads the saved entries once at startup, then writes changes in
the background every `--persist-interval` and once more on shutdown. If Redis
is unavailable at startup, the exporter starts with an empty cache and logs a
warning. If it goes down later, the exporter keeps working and retries the
writes.

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

See [charts/domain-expiry-exporter/values.yaml](./charts/domain-expiry-exporter/values.yaml)
for all options.

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
