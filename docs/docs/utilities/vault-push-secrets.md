---
title: vault-push-secrets
---

# vault-push-secrets

Periodically exports Vault secrets, roles, and policies to a Bitwarden Secrets Manager secret. Its purpose is enabling rapid cluster recovery: a single Bitwarden secret contains everything needed to restore Vault to its previous state.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `config.vaultAddr` | `""` | Vault API address (e.g. `http://vault:8200`) |
| `config.vaultAuthMount` | `""` | Kubernetes auth mount path (e.g. `kubernetes`) |
| `config.vaultAuthRole` | `""` | Kubernetes auth role to assume |
| `config.vaultSecretsMount` | `""` | Secret engine mount path to export (e.g. `secret`) |
| `config.bitwardenAccessTokenSecret` | `""` | Name of the Kubernetes Secret containing the Bitwarden access token |
| `config.bitwardenAccessTokenKey` | `""` | Key within the Secret that holds the Bitwarden access token |
| `config.bitwardenSecretId` | `""` | ID of the Bitwarden secret to update with the export |
| `config.logLevel` | `"info"` | Log level: `debug`, `info`, `warn`, or `error` |
| `deployment.image.tag` | `""` | Container image tag; defaults to the chart version |
| `deployment.resources` | `null` | Kubernetes resource requests and limits |
| `serviceAccount.name` | `""` | Service account name; defaults to the chart name |
