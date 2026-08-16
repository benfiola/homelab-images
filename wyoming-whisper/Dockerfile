ARG WYOMING_WHISPER_VERSION=3.6.0
FROM docker.io/rhasspy/wyoming-whisper:${WYOMING_WHISPER_VERSION}

ARG TARGETARCH

COPY install-cuda.sh /install-cuda.sh
RUN <<EOF
set -e
/install-cuda.sh
rm -rf /install-cuda.sh
# ctranslate2 4.6.2 supports cuda 12.4
.venv/bin/pip3 install --force-reinstall ctranslate2==4.6.2
EOF
