FROM ghcr.io/benfiola/homelab-images/lvm2:1.1.0 AS lvm2

FROM debian:trixie-slim@sha256:f6b08b09a05725f9431502d3b5714f5c4576e82c1a84fa1debe0c24a7e0c3ec4

ARG TARGETARCH

RUN <<EOF
set -e

apt-get update
apt-get install -y tar thin-provisioning-tools
rm -rf /var/lib/apt/lists/*
EOF

COPY --from=lvm2 /lvm2.tar.gz /tmp/lvm2.tar.gz
RUN <<EOF
tar xvzf /tmp/lvm2.tar.gz -C /
ln -fs /sbin/lvm /usr/sbin/lvm.static
rm -rf /tmp/lvm2.tar.gz
EOF

COPY .build/linstor-provision-disk_linux_${TARGETARCH} /usr/bin/linstor-provision-disk

ENTRYPOINT ["linstor-provision-disk"]
