---
title: vault-push-secrets
---

# vault-push-secrets

Periodically exports Vault secrets, roles, and policies to an age-encrypted YAML file stored in Google Cloud Storage. Its purpose is enabling rapid cluster recovery: during bootstrap, a single GCS file contains everything needed to restore Vault to its previous state.

## Helm Chart Values

| Value | Default | Description |
|-------|---------|-------------|
| `config.vaultAddr` | `""` | Vault API address (e.g. `http://vault:8200`) |
| `config.vaultAuthMount` | `""` | Kubernetes auth mount path (e.g. `kubernetes`) |
| `config.vaultAuthRole` | `""` | Kubernetes auth role to assume |
| `config.vaultSecretsMount` | `""` | Secret engine mount path to export (e.g. `secret`) |
| `config.gcsDestination` | `""` | GCS path for the export (e.g. `gs://bucket/path/vault-export.age`) |
| `config.gcsCredentialsSecret` | `""` | Name of the Kubernetes Secret containing GCS credentials |
| `config.gcsCredentialsKey` | `""` | Key within the Secret that holds the GCS credentials JSON |
| `config.encryptionKey` | `""` | age X25519 public key used to encrypt the export |
| `config.logLevel` | `"info"` | Log level: `debug`, `info`, `warn`, or `error` |
| `deployment.image.tag` | `""` | Container image tag; defaults to the chart version |
| `deployment.resources` | `null` | Kubernetes resource requests and limits |
| `serviceAccount.name` | `""` | Service account name; defaults to the chart name |
