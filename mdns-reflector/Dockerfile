FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/mdns-reflector_linux_${TARGETARCH} /usr/bin/mdns-reflector

ENTRYPOINT ["mdns-reflector"]
