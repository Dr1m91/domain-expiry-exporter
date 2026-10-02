# domain-expiry-exporter

RDAP-first Prometheus exporter for domain expiry monitoring, built to scale
to thousands of domains without hitting registrar rate limits.

Domain checks run entirely in a background scheduler, decoupled from
Prometheus scrapes. `/metrics` and `/probe` only ever read from an in-memory
registry — they never make a network call, so scrape latency and RDAP/WHOIS
availability are fully independent of each other.

## Features

- RDAP-first probing with automatic WHOIS fallback
- Adaptive check intervals: domains close to expiry are checked more often,
  domains far from expiry are checked less often
- Exponential backoff on repeated probe failures
- Bounded-concurrency background scheduler with per-cycle domain scanning
- Non-blocking `/probe` and `/metrics` endpoints (blackbox-exporter compatible)
- Automatic eviction of domains no longer being scraped
- Works with a static `domains.yaml` config, or dynamically via Prometheus
  scrape targets

## Usage

domain-expiry-exporter
--bind=:9222
--scan-interval=1m
--stale-after=24h
--concurrency=20


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

Or seed a static list at startup with `--config=domains.yaml`:

```yaml
domains:
  - example.com
  - another-example.com
```

## Helm chart

```bash
helm install domain-expiry-exporter oci://ghcr.io/dr1m91/charts/domain-expiry-exporter --version <version>
```

See [charts/domain-expiry-exporter/values.yaml](./charts/domain-expiry-exporter/values.yaml) for configuration options.

## Metrics

| Metric | Description |
|---|---|
| `domain_expiry_days` | Days until the domain expires |
| `domain_probe_success` | Whether the domain has ever been successfully probed |
| `domain_last_success_timestamp_seconds` | Unix timestamp of the last successful probe |
| `domain_consecutive_failures` | Number of consecutive failed probe attempts since the last success |

## Roadmap

- [ ] Pluggable storage backend (Redis, for shared state across replicas)

## Origin

Started from [caarlos0/domain_exporter](https://github.com/caarlos0/domain_exporter)
(archived August 2026), which probed WHOIS/RDAP synchronously on every scrape —
workable for a handful of domains, but not for fleets in the thousands. This
project keeps the original RDAP/WHOIS probing logic and rebuilds the caching
and scheduling layer around it.

## License

MIT — see [LICENSE](./LICENSE). Portions derived from
[caarlos0/domain_exporter](https://github.com/caarlos0/domain_exporter) (MIT, 2017).
