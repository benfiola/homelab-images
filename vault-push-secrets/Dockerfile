FROM debian:bookworm-slim

ARG BWS_VERSION="2.1.0"
ARG TARGETARCH

RUN <<EOF
apt -y update
apt -y install ca-certificates curl procps unzip
rm -rf /var/lib/apt/lists/*
EOF

# Fetch bws CLI binary
RUN <<EOF
set -e

ARCH=${TARGETARCH}
if [ "$ARCH" = "amd64" ]; then
  BWS_ARCH="x86_64"
elif [ "$ARCH" = "arm64" ]; then
  BWS_ARCH="aarch64"
else
  echo "Unsupported architecture: $ARCH"
  exit 1
fi


curl -fsSL "https://github.com/bitwarden/sdk-sm/releases/download/bws-v${BWS_VERSION}/bws-${BWS_ARCH}-unknown-linux-gnu-${BWS_VERSION}.zip" -o /tmp/archive.zip
unzip -q /tmp/archive.zip -d /usr/local/bin/
chmod +x /usr/local/bin/bws
rm /tmp/archive.zip
EOF

COPY .build/vault-push-secrets_linux_${TARGETARCH} /usr/bin/vault-push-secrets

ENTRYPOINT ["vault-push-secrets"]
