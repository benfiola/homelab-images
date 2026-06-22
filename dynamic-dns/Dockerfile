FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/dynamic-dns /usr/bin/dynamic-dns

ENTRYPOINT ["dynamic-dns"]
