FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
apt -y update
apt -y install procps
rm -rf /var/lib/apt/lists/*
EOF

COPY .build/vault-push-secrets_linux_${TARGETARCH} /usr/bin/vault-push-secrets

ENTRYPOINT ["vault-push-secrets"]
