FROM debian:trixie-slim

ARG TARGETARCH

RUN <<EOF
apt-get update
apt-get install -y ca-certificates curl libarchive-tools rsync vim
rm -rf /var/lib/apt/lists/*
EOF
