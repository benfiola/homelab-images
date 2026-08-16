ARG WYOMING_WHISPER_VERSION=3.6.0
FROM docker.io/rhasspy/wyoming-whisper:${WYOMING_WHISPER_VERSION}

ARG TARGETARCH

COPY install-cuda-12.sh /install-cuda-12.sh
RUN <<EOF
set -e
/install-cuda-12.sh
rm -rf /install-cuda-12.sh
.venv/bin/pip3 install --force-reinstall ctranslate2==4.4.0
EOF
