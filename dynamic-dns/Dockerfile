FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
apt -y update
apt -y install ca-certificates
rm -rf /var/lib/apt/lists/*
EOF

COPY .build/dynamic-dns_linux_${TARGETARCH} /usr/bin/dynamic-dns

ENTRYPOINT ["dynamic-dns"]
