FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/vault-unseal /usr/bin/vault-unseal

ENTRYPOINT ["vault-unseal"]
