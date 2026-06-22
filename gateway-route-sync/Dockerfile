FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/gateway-route-sync_linux_${TARGETARCH} /usr/bin/gateway-route-sync

ENTRYPOINT ["gateway-route-sync"]
