ARG ADVANCED_CAMERA_CARD_VERSION=7.27.4
ARG FRIGATE_VERSION=5.15.4
ARG HA_VERSION=2026.6.4

FROM alpine AS advanced-camera-card-addon

ARG ADVANCED_CAMERA_CARD_VERSION

COPY install-advanced-camera-card.sh /install.sh
RUN <<EOF
set -e

/install.sh "${ADVANCED_CAMERA_CARD_VERSION}" /addons/www/advanced-camera-card
rm -rf /install.sh
EOF

FROM alpine AS frigate-addon

ARG FRIGATE_VERSION

COPY install-frigate.sh /install.sh
RUN <<EOF
set -e

/install.sh "${FRIGATE_VERSION}" /addons/custom_components/frigate
rm -rf /install.sh
EOF

FROM ghcr.io/home-assistant/home-assistant:${HA_VERSION}

ARG TARGETARCH

COPY --from=advanced-camera-card-addon /addons/www/advanced-camera-card /addons/www/advanced-camera-card
COPY --from=frigate-addon /addons/custom_components/frigate /addons/custom_components/frigate
COPY rootfs/ /

COPY install-addon-requirements.py /install.py
RUN <<EOF
set -e

python3 /install.py /addons/custom_components
rm /install.py
EOF
