FROM debian:trixie-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2

RUN <<'EOF'
set -e
apt-get update
apt-get install -y --no-install-recommends avahi-daemon
rm -rf /var/lib/apt/lists/*
EOF

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
