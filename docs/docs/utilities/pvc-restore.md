---
title: pvc-restore
---

# pvc-restore

A Kubernetes controller that automates PVC restoration from volsync backups. Annotate a PVC with a snapshot selector and the controller handles everything: scaling down workloads that use the PVC, syncing backup data via a temporary ReplicationDestination, scaling workloads back up, and cleaning up temporary resources.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `config.cacheStorageClass` | `""` | Storage class for ReplicationDestination cache volumes; defaults to the cluster default |
| `deployment.image.tag` | `""` | Container image tag; defaults to the chart version |
| `deployment.replicas` | `1` | Number of controller replicas |
| `deployment.resources` | `null` | Kubernetes resource requests and limits |
