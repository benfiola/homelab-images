FROM golang:1.25.8 as builder

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/vault-unseal /usr/bin/vault-unseal

ENTRYPOINT ["vault-unseal"]
