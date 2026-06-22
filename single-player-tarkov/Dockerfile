FROM debian:bookworm-slim AS dotnet_installer

ARG TARGETARCH
ARG DOTNET_VERSION=9.0.312

COPY install-dotnet.sh /install.sh
RUN /install.sh "${DOTNET_VERSION}" "${TARGETARCH}" /out

FROM debian:bookworm-slim

ARG TARGETARCH

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
COPY .build/single-player-tarkov_linux_${TARGETARCH} /usr/bin/single-player-tarkov

USER game

EXPOSE 6969/tcp
EXPOSE 8880/tcp

ENTRYPOINT [ "single-player-tarkov" ]
