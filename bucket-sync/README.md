# bucket-sync

A Kubernetes controller that automates periodic synchronization between object storage buckets using scheduled or on-demand rclone-based sync jobs. It manages the complete lifecycle of each sync operation — distributed locking to prevent concurrent runs, Kubernetes Job creation and monitoring, and resource cleanup — allowing bucket syncs to be managed declaratively alongside other cluster infrastructure.

## Helm Chart Values

| Value                  | Default | Description                                        |
| ---------------------- | ------- | -------------------------------------------------- |
| `deployment.image.tag` | `""`    | Container image tag; defaults to the chart version |
| `deployment.replicas`  | `1`     | Number of controller replicas                      |
| `deployment.resources` | `null`  | Kubernetes resource requests and limits            |
| `metrics.podMonitor.enabled` | `false` | Create a Prometheus Operator `PodMonitor` that scrapes the controller |
| `metrics.podMonitor.interval` | `""` | Scrape interval (e.g. `30s`); defaults to the Prometheus setting |
| `metrics.podMonitor.scrapeTimeout` | `""` | Scrape timeout (e.g. `10s`) |
| `metrics.podMonitor.labels` | `{}` | Extra labels on the `PodMonitor` (e.g. `release: prometheus`) so Prometheus selects it |
| `metrics.dashboard.enabled` | `false` | Create a ConfigMap containing a Grafana dashboard for the Grafana sidecar to load |
| `metrics.dashboard.labels` | `{grafana_dashboard: "1"}` | Labels on the dashboard ConfigMap; must match the Grafana sidecar label selector |
| `metrics.dashboard.annotations` | `{}` | Annotations on the dashboard ConfigMap (e.g. `grafana_folder: Homelab`) |
| `metrics.alerts.enabled` | `false` | Create a Prometheus Operator `PrometheusRule` with alerts for this component |
| `metrics.alerts.labels` | `{}` | Extra labels on the `PrometheusRule` (e.g. `release: prometheus`) so Prometheus selects it |
| `metrics.alerts.consecutiveFailureThreshold` | `3` | Consecutive failed syncs of a policy before `BucketSyncRepeatedlyFailing` (critical) fires; the count resets on any success |
