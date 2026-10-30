ARG HA_VERSION=2026.10.4

FROM alpine:3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS addons

COPY install-*.sh /

RUN <<EOF
set -e
mkdir -p /addons/www /addons/custom_components /addons/blueprints/automation /addons/blueprints/script
for script in /install-*.sh; do ADDONS_DIR=/addons "$script"; done
rm /install-*.sh
EOF

FROM ghcr.io/home-assistant/home-assistant:${HA_VERSION}

ARG TARGETARCH

COPY --from=addons /addons /addons
COPY rootfs/ /

COPY patches/ /patches/
COPY patch-*.sh /

RUN <<EOF
set -e
for script in /patch-*.sh; do "$script"; done
rm -r /patch-*.sh /patches
EOF

COPY install-addon-requirements.py /install.py
RUN <<EOF
set -e

python3 /install.py /addons/custom_components
rm /install.py
EOF
