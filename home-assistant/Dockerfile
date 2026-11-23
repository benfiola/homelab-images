ARG HA_VERSION=2026.11.3@sha256:24e541ddae5818bf8eb27898b9ba00d0c344a655aa5097e9e9bd4b19356be847

FROM alpine:3@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS addons

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
