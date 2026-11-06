FROM ghcr.io/benfiola/homelab-images/lvm2:1.1.2@sha256:8b3c134e724155be213243027853f21faf4dfcf22617890c593c82695a109776 AS lvm2

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
ln -fs /usr/sbin/lvm.static /sbin/lvm
rm -rf /tmp/lvm2.tar.gz
EOF

COPY .build/linstor-provision-disk_linux_${TARGETARCH} /usr/bin/linstor-provision-disk

ENTRYPOINT ["linstor-provision-disk"]
