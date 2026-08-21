---
title: mdns-reflector
---

# mdns-reflector

A Kubernetes DaemonSet that reflects mDNS (Multicast DNS) packets between network interfaces on each node, enabling service discovery to work across network segments that would otherwise be isolated. Reflection is bidirectional — any interface can both receive and forward packets to all other interfaces. By default, all non-loopback interfaces that are up and have an IP address are used; a specific subset can be provided via `config.interfaces` when only certain interfaces should participate.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `config.interfaces` | `""` | Comma-separated interfaces to reflect mDNS packets between; defaults to all non-loopback interfaces that are up |
| `config.logLevel` | `""` | Log level: `debug`, `info`, `warn`, or `error` |
| `config.logFormat` | `""` | Log format: `text` or `json` |
| `daemonSet.image.tag` | `""` | Container image tag; defaults to the chart version |
| `daemonSet.resources` | `null` | Kubernetes resource requests and limits |
| `daemonSet.hostNetwork` | `false` | Use the host network namespace to access all host interfaces |
| `serviceAccount.name` | `""` | Service account name; defaults to the chart name |
