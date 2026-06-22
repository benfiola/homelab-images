FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/mdns-reflector /usr/bin/mdns-reflector

ENTRYPOINT ["mdns-reflector"]
