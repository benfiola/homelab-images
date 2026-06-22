FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/dynamic-dns_linux_${TARGETARCH} /usr/bin/dynamic-dns

ENTRYPOINT ["dynamic-dns"]
