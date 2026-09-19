# domain-expiry-exporter

RDAP-first Prometheus exporter for domain expiry monitoring, built for fleets
of hundreds to thousands of domains without hitting registrar/RDAP rate limits.

## Origin

This project started as a fork of [caarlos0/domain_exporter](https://github.com/caarlos0/domain_exporter),
which was archived by its author in August 2026. The original exporter probes
WHOIS/RDAP synchronously on every Prometheus scrape, which does not scale past
a few hundred domains without hitting `i/o timeout` errors on the WHOIS/RDAP
side. This project decouples probing from scraping via a background scheduler
and adaptive polling intervals.

## Status

Early development (v0.1.0 scope): background poller, in-memory cache,
non-blocking `/metrics`. See CHANGELOG.md / releases for progress.

## License

MIT — see [LICENSE](./LICENSE). Portions derived from
[caarlos0/domain_exporter](https://github.com/caarlos0/domain_exporter) (MIT, 2017).
