---
title: dynamic-dns
---

# dynamic-dns

Periodically detects your public IP address and updates Cloudflare DNS A records to keep them in sync with your dynamic IP. Designed to run in a restricted Kubernetes pod security context — no root access, no filesystem writes, minimal capabilities — making it a safe drop-in replacement for traditional dynamic DNS tools like `ddclient`.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `config.cloudflareApiTokenSecret` | `""` | Name of the Kubernetes Secret containing the Cloudflare API token |
| `config.cloudflareApiTokenKey` | `""` | Key within the Secret that holds the API token |
| `config.cloudflareZones` | `[]` | List of Cloudflare zones and their domains to keep in sync |
| `config.interval` | `""` | How often to check and update DNS records (e.g. `10m`) |
| `config.logLevel` | `""` | Log level: `debug`, `info`, `warn`, or `error` |
| `config.logFormat` | `""` | Log format: `text` or `json` |
| `deployment.image.tag` | `""` | Container image tag; defaults to the chart version |
| `deployment.resources` | `null` | Kubernetes resource requests and limits |
| `serviceAccount.name` | `""` | Service account name; defaults to the chart name |
