FROM golang:1.25.8 as builder

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/vault-push-secrets /usr/bin/vault-push-secrets

ENTRYPOINT ["vault-push-secrets"]
