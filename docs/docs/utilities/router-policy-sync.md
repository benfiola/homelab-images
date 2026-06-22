---
title: router-policy-sync
---

# router-policy-sync

A Kubernetes controller that synchronizes Cilium network policies with MikroTik router firewall rules. When a `CiliumClusterwideNetworkPolicy` is annotated for sync, the controller automatically extracts the allowed source IPs, discovers the gateway IP from the matching LoadBalancer service, and keeps the corresponding MikroTik firewall filters, NAT rules, and address lists up to date.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `config.mikrotikBaseUrl` | `""` | Base URL of the MikroTik REST API (e.g. `https://192.168.1.1/rest`) |
| `config.mikrotikUsername` | `""` | Username for MikroTik API authentication |
| `config.mikrotikPasswordSecret` | `""` | Name of the Kubernetes Secret containing the MikroTik password |
| `config.mikrotikPasswordKey` | `""` | Key within the Secret that holds the password |
| `config.reservedCIDRs` | `[]` | List of CIDR ranges to treat as reserved (rejected from sync) |
| `config.syncInterval` | `""` | How often to reconcile firewall rules to detect drift (e.g. `5m`) |
| `deployment.image.tag` | `""` | Container image tag; defaults to the chart version |
| `deployment.replicas` | `1` | Number of controller replicas |
| `deployment.resources` | `null` | Kubernetes resource requests and limits |
