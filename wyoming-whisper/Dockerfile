ARG WYOMING_WHISPER_VERSION=3.8.1@sha256:ba6fcb6056ebe237d15a325381763a80af2fc8fcedaa04f6a714a7375eb20d80
FROM docker.io/rhasspy/wyoming-whisper:${WYOMING_WHISPER_VERSION}

ARG TARGETARCH

COPY install-cuda.sh /install-cuda.sh
RUN <<EOF
set -e
/install-cuda.sh
rm -rf /install-cuda.sh
# renovate-pinned: depName=ctranslate2 currentValue=4.6.2 reason=release that supports CUDA 12.4
# ctranslate2 4.6.2 supports cuda 12.4
.venv/bin/pip3 install --force-reinstall ctranslate2==4.6.2
EOF
