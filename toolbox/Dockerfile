FROM debian:trixie-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2

ARG TARGETARCH

RUN <<EOF
apt-get update
apt-get install -y ca-certificates curl libarchive-tools rsync vim
rm -rf /var/lib/apt/lists/*
EOF
