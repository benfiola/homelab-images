FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/vault-push-secrets_linux_${TARGETARCH} /usr/bin/vault-push-secrets

ENTRYPOINT ["vault-push-secrets"]
