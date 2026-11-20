ARG WYOMING_WHISPER_VERSION=3.9.0@sha256:3cc01f646c8b149fe47b2a5e8bfaf31866e1bc18faffadb26c9b8be58cbe5703
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
