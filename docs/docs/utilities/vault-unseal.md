---
title: vault-unseal
---

# vault-unseal

A sidecar container that automatically unseals Vault whenever its pod starts, eliminating the need for manual `vault unseal` commands after pod restarts, node maintenance, or cluster failures. It polls until Vault is reachable, checks the seal status, and unseals using a key file provisioned by an init container, Kubernetes Secret, or persistent volume.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VAULT_UNSEAL_KEY` | Yes | — | Path to the file containing the Vault unseal key |
| `VAULT_ADDR` | No | — | Vault API address (e.g. `http://127.0.0.1:8200`) |
| `CONTINUOUS` | No | — | If set, keep running and re-unseal if Vault becomes sealed again |
