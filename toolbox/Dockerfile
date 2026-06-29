FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
apt-get update
apt-get install -y ca-certificates curl libarchive-tools
rm -rf /var/lib/apt/lists/*
EOF
