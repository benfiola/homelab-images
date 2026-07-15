---
title: vault-auth-proxy
---

# vault-auth-proxy

A sidecar container that enables unauthenticated localhost access to Vault by proxying requests and automatically injecting the root token as the `X-Vault-Token` header. Applications point their Vault client at localhost:8201 and the proxy handles authentication transparently.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LISTEN_ADDR` | No | `127.0.0.1:8201` | Address the proxy listens on (e.g. `0.0.0.0:8201` for Kubernetes) |
| `VAULT_ADDR` | No | `http://localhost:8200` | Vault API address to proxy requests to |
| `ROOT_TOKEN_PATH` | No | `/vault/data/root-token` | Path to file containing the root token |
