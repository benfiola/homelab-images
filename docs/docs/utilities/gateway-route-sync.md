---
title: gateway-route-sync
---

# gateway-route-sync

A Kubernetes controller that automatically generates Gateway listeners from the hostnames and protocols declared in attached routes. By keeping listeners in sync with routes, it eliminates manual listener management and ensures cert-manager can derive TLS certificates for every hostname — even as routes are added or changed.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `deployment.image.tag` | `""` | Container image tag; defaults to the chart version |
| `deployment.replicas` | `1` | Number of controller replicas |
| `deployment.resources` | `null` | Kubernetes resource requests and limits |
