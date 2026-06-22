FROM debian:bookworm-slim AS dotnet_installer

ARG TARGETARCH
ARG DOTNET_VERSION=9.0.312

RUN <<EOF
set -e

export DEBIAN_FRONTEND=noninteractive
apt -y update
apt -y install curl libc6 libgcc-s1 libgssapi-krb5-2 libicu72 libssl3 libstdc++6 zlib1g
rm -rf /var/lib/apt/lists/*

ARCH="${TARGETARCH}"
if [ "${ARCH}" = "amd64" ]; then 
  ARCH="x64"
fi

curl -fsSL -o archive.tar.gz "https://builds.dotnet.microsoft.com/dotnet/Sdk/${DOTNET_VERSION}/dotnet-sdk-${DOTNET_VERSION}-linux-${ARCH}.tar.gz"
mkdir -p /out
tar -C /out -xzf archive.tar.gz
EOF

FROM debian:bookworm-slim

RUN <<EOF
set -e

export DEBIAN_FRONTEND=noninteractive
apt -y update
apt -y install curl libarchive-tools squashfs-tools nodejs
rm -rf /var/lib/apt/lists/*

useradd -m -u 1000 game
mkdir -p /cache /data /game
chown -R game:game /cache /data /game
EOF

COPY --from=dotnet_installer /out/dotnet /usr/share/dotnet
RUN ln -s /usr/share/dotnet/dotnet /usr/local/bin/dotnet
COPY $TARGETPLATFORM/single-player-tarkov /usr/bin/single-player-tarkov

USER game

EXPOSE 6969/tcp
EXPOSE 8880/tcp

ENTRYPOINT [ "single-player-tarkov" ]
