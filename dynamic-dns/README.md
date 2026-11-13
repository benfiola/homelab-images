# dynamic-dns

Periodically detects your public IP address and updates Cloudflare DNS A records to keep them in sync with your dynamic IP. Designed to run in a restricted Kubernetes pod security context — no root access, no filesystem writes, minimal capabilities — making it a safe drop-in replacement for traditional dynamic DNS tools like `ddclient`.

## Helm Chart Values

| Value                             | Default | Description                                                       |
| --------------------------------- | ------- | ----------------------------------------------------------------- |
| `config.cloudflareApiTokenSecret` | `""`    | Name of the Kubernetes Secret containing the Cloudflare API token |
| `config.cloudflareApiTokenKey`    | `""`    | Key within the Secret that holds the API token                    |
| `config.cloudflareZones`          | `[]`    | List of Cloudflare zones and their domains to keep in sync        |
| `config.interval`                 | `""`    | How often to check and update DNS records (e.g. `10m`)            |
| `config.logLevel`                 | `""`    | Log level: `debug`, `info`, `warn`, or `error`                    |
| `config.logFormat`                | `""`    | Log format: `text` or `json`                                      |
| `deployment.image.tag`            | `""`    | Container image tag; defaults to the chart version                |
| `deployment.resources`            | `null`  | Kubernetes resource requests and limits                           |
| `serviceAccount.name`             | `""`    | Service account name; defaults to the chart name                  |
| `metrics.podMonitor.enabled` | `false` | Create a Prometheus Operator `PodMonitor` that scrapes the controller |
| `metrics.podMonitor.interval` | `""` | Scrape interval (e.g. `30s`); defaults to the Prometheus setting |
| `metrics.podMonitor.scrapeTimeout` | `""` | Scrape timeout (e.g. `10s`) |
| `metrics.podMonitor.labels` | `{}` | Extra labels on the `PodMonitor` (e.g. `release: prometheus`) so Prometheus selects it |
| `metrics.dashboard.enabled` | `false` | Create a ConfigMap containing a Grafana dashboard for the Grafana sidecar to load |
| `metrics.dashboard.labels` | `{grafana_dashboard: "1"}` | Labels on the dashboard ConfigMap; must match the Grafana sidecar label selector |
| `metrics.dashboard.annotations` | `{}` | Annotations on the dashboard ConfigMap (e.g. `grafana_folder: Homelab`) |
| `metrics.alerts.enabled` | `false` | Create a Prometheus Operator `PrometheusRule` with alerts for this component |
| `metrics.alerts.labels` | `{}` | Extra labels on the `PrometheusRule` (e.g. `release: prometheus`) so Prometheus selects it |
| `metrics.alerts.staleAfterSeconds` | `3600` | Seconds without a successful sync before `DynamicDNSSyncStale` fires; should exceed a few intervals |

## Metrics

In continuous mode, Prometheus metrics are served on `:8080/metrics` (override with `METRICS_ADDRESS`).

| Metric                                              | Description                                    |
| --------------------------------------------------- | ---------------------------------------------- |
| `dynamic_dns_syncs_total{result}`                   | Sync attempts (`success` / `failure`)          |
| `dynamic_dns_sync_duration_seconds`                 | Sync duration histogram                        |
| `dynamic_dns_sync_errors_total{stage}`              | Errors by stage (public IP, zone/record lookup, update) |
| `dynamic_dns_record_updates_total`                  | DNS records changed to a new IP                |
| `dynamic_dns_last_successful_sync_timestamp_seconds` | Unix time of the last fully successful sync   |
