FROM debian:trixie-slim@sha256:f6b08b09a05725f9431502d3b5714f5c4576e82c1a84fa1debe0c24a7e0c3ec4

ARG TARGETARCH

RUN <<EOF
apt-get update
apt-get install -y ca-certificates curl libarchive-tools rsync vim
rm -rf /var/lib/apt/lists/*
EOF
