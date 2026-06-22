FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
set -e

apt-get update
apt-get install -y thin-provisioning-tools
rm -rf /var/lib/apt/lists/*
EOF

COPY $TARGETPLATFORM/gateway-route-sync /usr/bin/gateway-route-sync

ENTRYPOINT ["linstor-provision-disk"]
