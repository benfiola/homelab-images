ARG MUSIC_ASSISTANT_VERSION=2.9.13

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
