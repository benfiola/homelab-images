ARG FRIGATE_VERSION=5.15.4
ARG HA_VERSION=2026.6.4

FROM alpine AS frigate-integration

ARG FRIGATE_VERSION

COPY install-frigate-integration.sh /install.sh
RUN <<EOF
/install.sh "${FRIGATE_VERSION}" /custom_components
rm -rf /install.sh
EOF

FROM ghcr.io/home-assistant/home-assistant:${HA_VERSION}

ARG TARGETARCH

COPY --from=frigate-integration /custom_components/frigate /custom_components/frigate
COPY rootfs/ /

COPY install-custom-component-requirements.py /install.py
RUN <<EOF
set -e

python3 /install.py /custom_components
rm /install.py
EOF
