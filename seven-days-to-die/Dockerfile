FROM debian:bookworm-slim AS depotdownloader_installer

ARG TARGETARCH
ARG DEPOTDOWNLOADER_VERSION=3.4.0

COPY install-depotdownloader.sh /install.sh
RUN /install.sh "${DEPOTDOWNLOADER_VERSION}" "${TARGETARCH}" /out

FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

RUN <<EOF
set -e

export DEBIAN_FRONTEND=noninteractive
apt -y update
apt -y install ca-certificates libarchive-tools libssl3 squashfs-tools
rm -rf /var/lib/apt/lists/*

useradd -m -u 1000 game
mkdir -p /cache /data /game
chown -R game:game /cache /data /game
EOF

COPY --from=depotdownloader_installer /out/DepotDownloader /usr/bin/DepotDownloader
COPY .build/seven-days-to-die_linux_${TARGETARCH} /usr/bin/seven-days-to-die

USER game

EXPOSE 8080/tcp
EXPOSE 8081/tcp
EXPOSE 8880/tcp
EXPOSE 26900/udp
EXPOSE 26900/tcp
EXPOSE 26901/udp
EXPOSE 26902/udp
EXPOSE 26903/udp

ENTRYPOINT [ "seven-days-to-die" ]
