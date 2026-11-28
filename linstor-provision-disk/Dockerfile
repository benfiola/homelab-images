FROM ghcr.io/benfiola/homelab-images/lvm2:1.1.4@sha256:3a1e3c1143bbf7ffd9e6a5f416e8a2b656a8adbab0b05c1c23cab473c7cc08be AS lvm2

FROM debian:trixie-slim@sha256:8aa5a7bf5a1955bddbe7d74fcdd0b91ab30e74362a8b73af672b0481b35b44a0

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
