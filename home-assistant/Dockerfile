ARG HA_VERSION=2026.6.4

FROM alpine AS addons

COPY install-*.sh /

RUN <<EOF
set -e
mkdir -p /addons/www /addons/custom_components
for script in /install-*.sh; do ADDONS_DIR=/addons "$script"; done
rm /install-*.sh
EOF

FROM ghcr.io/home-assistant/home-assistant:${HA_VERSION}

ARG TARGETARCH

COPY --from=addons /addons /addons
COPY rootfs/ /

COPY install-addon-requirements.py /install.py
RUN <<EOF
set -e

python3 /install.py /addons/custom_components
rm /install.py
EOF
