FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/vault-auth-proxy_linux_${TARGETARCH} /usr/bin/vault-auth-proxy

ENTRYPOINT ["vault-auth-proxy"]
