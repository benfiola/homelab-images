FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
set -e

apt-get update
apt-get install -y thin-provisioning-tools
rm -rf /var/lib/apt/lists/*
EOF

COPY .build/linstor-provision-disk_linux_${TARGETARCH} /usr/bin/linstor-provision-disk

ENTRYPOINT ["linstor-provision-disk"]
