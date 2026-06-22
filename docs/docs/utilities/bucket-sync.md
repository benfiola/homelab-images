---
title: bucket-sync
---

# bucket-sync

A Kubernetes controller that automates periodic synchronization between object storage buckets using scheduled or on-demand rclone-based sync jobs. It manages the complete lifecycle of each sync operation — distributed locking to prevent concurrent runs, Kubernetes Job creation and monitoring, and resource cleanup — allowing bucket syncs to be managed declaratively alongside other cluster infrastructure.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `deployment.image.tag` | `""` | Container image tag; defaults to the chart version |
| `deployment.replicas` | `1` | Number of controller replicas |
| `deployment.resources` | `null` | Kubernetes resource requests and limits |
