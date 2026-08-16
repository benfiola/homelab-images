ARG WYOMING_WHISPER_VERSION=3.6.0
FROM docker.io/rhasspy/wyoming-whisper:${WYOMING_WHISPER_VERSION}

ARG TARGETARCH

COPY install-cuda.sh /install-cuda.sh
RUN <<EOF
set -e
/install-cuda.sh
rm -rf /install-cuda.sh
EOF
