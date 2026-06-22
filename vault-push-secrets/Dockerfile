FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/vault-push-secrets /usr/bin/vault-push-secrets

ENTRYPOINT ["vault-push-secrets"]
