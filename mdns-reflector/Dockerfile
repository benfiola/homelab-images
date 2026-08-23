FROM debian:bookworm-slim

RUN <<'EOF'
set -e
apt-get update
apt-get install -y --no-install-recommends avahi-daemon
rm -rf /var/lib/apt/lists/*
EOF

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
