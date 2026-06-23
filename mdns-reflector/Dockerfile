FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
apt -y update
apt -y install procps
rm -rf /var/lib/apt/lists/*
EOF

COPY .build/mdns-reflector_linux_${TARGETARCH} /usr/bin/mdns-reflector

ENTRYPOINT ["mdns-reflector"]
