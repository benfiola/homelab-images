FROM debian:trixie-slim@sha256:8aa5a7bf5a1955bddbe7d74fcdd0b91ab30e74362a8b73af672b0481b35b44a0

RUN <<'EOF'
set -e
apt-get update
apt-get install -y --no-install-recommends avahi-daemon
rm -rf /var/lib/apt/lists/*
EOF

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
