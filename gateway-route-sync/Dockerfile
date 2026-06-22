FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/gateway-route-sync /usr/bin/gateway-route-sync

ENTRYPOINT ["gateway-route-sync"]
