ARG WYOMING_WHISPER_VERSION=3.9.1@sha256:ac3fdbdb4d6d7c5899634a63cfe4bac78634d27c7a1b061b2bb5d8f1054853b5
FROM docker.io/rhasspy/wyoming-whisper:${WYOMING_WHISPER_VERSION}

ARG TARGETARCH

COPY install-cuda.sh /install-cuda.sh
RUN <<EOF
set -e
/install-cuda.sh
rm -rf /install-cuda.sh
# ctranslate2 4.6.2 supports cuda 12.4
# renovate-pinned: depName=ctranslate2 reason=release that supports CUDA 12.4
.venv/bin/pip3 install --force-reinstall ctranslate2==4.6.2
EOF
