FROM debian:trixie-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2

ARG TARGETARCH

RUN <<EOF
apt -y update
apt -y install ca-certificates procps
rm -rf /var/lib/apt/lists/*
EOF

COPY .build/dynamic-dns_linux_${TARGETARCH} /usr/bin/dynamic-dns

ENTRYPOINT ["dynamic-dns"]
