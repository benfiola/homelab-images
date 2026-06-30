FROM ghcr.io/benfiola/homelab-images/lvm2:1.0.6 AS lvm2

FROM debian:bookworm-slim

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
