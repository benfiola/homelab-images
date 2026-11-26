FROM debian:trixie-slim@sha256:8aa5a7bf5a1955bddbe7d74fcdd0b91ab30e74362a8b73af672b0481b35b44a0

ARG TARGETARCH

RUN <<EOF
apt-get update
apt-get install -y ca-certificates curl libarchive-tools rsync vim
rm -rf /var/lib/apt/lists/*
EOF
