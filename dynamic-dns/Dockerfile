FROM debian:trixie-slim@sha256:8aa5a7bf5a1955bddbe7d74fcdd0b91ab30e74362a8b73af672b0481b35b44a0

ARG TARGETARCH

RUN <<EOF
apt -y update
apt -y install ca-certificates procps
rm -rf /var/lib/apt/lists/*
EOF

COPY .build/dynamic-dns_linux_${TARGETARCH} /usr/bin/dynamic-dns

ENTRYPOINT ["dynamic-dns"]
