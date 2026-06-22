---
title: mdns-reflector
---

# mdns-reflector

A Kubernetes DaemonSet that forwards mDNS (Multicast DNS) broadcast packets between network interfaces on each node, enabling service discovery to work across network segments that would otherwise be isolated. It runs with minimal privileges (NET_RAW capability only) and supports automatic source interface detection from the default route.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `config.sourceInterfaces` | `""` | Comma-separated interfaces to listen on; auto-detected from the default route if omitted |
| `config.destInterfaces` | `""` | Comma-separated interfaces to forward mDNS packets to (required) |
| `config.logLevel` | `""` | Log level: `debug`, `info`, `warn`, or `error` |
| `config.logFormat` | `""` | Log format: `text` or `json` |
| `daemonSet.image.tag` | `""` | Container image tag; defaults to the chart version |
| `daemonSet.resources` | `null` | Kubernetes resource requests and limits |
| `daemonSet.hostNetwork` | `false` | Use the host network namespace to access all host interfaces |
| `serviceAccount.name` | `""` | Service account name; defaults to the chart name |
