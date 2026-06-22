ARG FRIGATE_VERSION=5.15.4
ARG HA_VERSION=2025.6.3

FROM alpine AS frigate-integration

ARG FRIGATE_VERSION

RUN <<EOF
apk add --no-cache curl 
curl -o archive.tar.gz -fsSL "https://github.com/blakeblackshear/frigate-hass-integration/archive/refs/tags/v${FRIGATE_VERSION}.tar.gz"
mkdir -p /custom_components
tar xz --strip-components=2 -C /custom_components "frigate-hass-integration-${FRIGATE_VERSION}/custom_components/frigate"
EOF

FROM ghcr.io/home-assistant/home-assistant:${HA_VERSION}

ARG TARGETARCH

COPY --from=frigate-integration /custom_components/ /custom_components/
COPY rootfs/ /
