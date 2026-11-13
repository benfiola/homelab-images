# pvc-restore

A Kubernetes controller that automates PVC restoration from volsync backups. Annotate a PVC with a snapshot selector and the controller handles everything: scaling down workloads that use the PVC, syncing backup data via a temporary ReplicationDestination, scaling workloads back up, and cleaning up temporary resources.

## Helm Chart Values

| Value                      | Default | Description                                                                             |
| -------------------------- | ------- | --------------------------------------------------------------------------------------- |
| `config.cacheStorageClass` | `""`    | Storage class for ReplicationDestination cache volumes; defaults to the cluster default |
| `deployment.image.tag`     | `""`    | Container image tag; defaults to the chart version                                      |
| `deployment.replicas`      | `1`     | Number of controller replicas                                                           |
| `deployment.resources`     | `null`  | Kubernetes resource requests and limits                                                 |
| `metrics.podMonitor.enabled` | `false` | Create a Prometheus Operator `PodMonitor` that scrapes the controller |
| `metrics.podMonitor.interval` | `""` | Scrape interval (e.g. `30s`); defaults to the Prometheus setting |
| `metrics.podMonitor.scrapeTimeout` | `""` | Scrape timeout (e.g. `10s`) |
| `metrics.podMonitor.labels` | `{}` | Extra labels on the `PodMonitor` (e.g. `release: prometheus`) so Prometheus selects it |
| `metrics.dashboard.enabled` | `false` | Create a ConfigMap containing a Grafana dashboard for the Grafana sidecar to load |
| `metrics.dashboard.labels` | `{grafana_dashboard: "1"}` | Labels on the dashboard ConfigMap; must match the Grafana sidecar label selector |
| `metrics.dashboard.annotations` | `{}` | Annotations on the dashboard ConfigMap (e.g. `grafana_folder: Homelab`) |
| `metrics.alerts.enabled` | `false` | Create a Prometheus Operator `PrometheusRule` with alerts for this component |
| `metrics.alerts.labels` | `{}` | Extra labels on the `PrometheusRule` (e.g. `release: prometheus`) so Prometheus selects it |
