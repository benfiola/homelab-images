ARG MUSIC_ASSISTANT_VERSION=2.10.14@sha256:7a2c25e78ed47ee2752c6b6e0667e561f9c40195dd1f017d0adf20dedb7ad8e0

FROM ghcr.io/music-assistant/server:${MUSIC_ASSISTANT_VERSION}

COPY base-url-override.patch /tmp/

RUN <<'EOF'
set -e
apt-get update
apt-get install -y --no-install-recommends patch
INSTALL_DIR=$(python3 -c "import music_assistant, os; print(os.path.dirname(os.path.dirname(music_assistant.__file__)))")
patch -p1 -d "$INSTALL_DIR" < /tmp/base-url-override.patch
rm /tmp/base-url-override.patch
apt-get purge -y --auto-remove patch
rm -rf /var/lib/apt/lists/*
EOF
