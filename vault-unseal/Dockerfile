FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/vault-unseal_linux_${TARGETARCH} /usr/bin/vault-unseal

ENTRYPOINT ["vault-unseal"]
