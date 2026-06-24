ARG ADVANCED_CAMERA_CARD_VERSION=7.27.4
ARG FRIGATE_VERSION=5.15.4
ARG HA_VERSION=2026.6.4

FROM alpine AS advanced-camera-card-integration

ARG ADVANCED_CAMERA_CARD_VERSION

COPY install-advanced-camera-card.sh /install.sh
RUN <<EOF
set -e

/install.sh "${ADVANCED_CAMERA_CARD_VERSION}" /custom_components/advanced-camera-card
rm -rf /install.sh
EOF

FROM alpine AS frigate-integration

ARG FRIGATE_VERSION

COPY install-frigate-integration.sh /install.sh
RUN <<EOF
set -e

/install.sh "${FRIGATE_VERSION}" /custom_components/frigate
rm -rf /install.sh
EOF

FROM ghcr.io/home-assistant/home-assistant:${HA_VERSION}

ARG TARGETARCH

COPY --from=advanced-camera-card-integration /custom_components/advanced-camera-card /custom_components/advanced-camera-card
COPY --from=frigate-integration /custom_components/frigate /custom_components/frigate
COPY rootfs/ /

COPY install-custom-component-requirements.py /install.py
RUN <<EOF
set -e

python3 /install.py /custom_components
rm /install.py
EOF
