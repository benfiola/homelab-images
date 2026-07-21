FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
apt -y update
apt -y install ffmpeg
rm -rf /var/lib/apt/lists/*
EOF

COPY .build/loudnorm_linux_${TARGETARCH} /usr/bin/loudnorm

ENTRYPOINT ["loudnorm"]
