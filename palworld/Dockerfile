FROM debian:bookworm-slim AS depotdownloader_installer

ARG TARGETARCH
ARG DEPOTDOWNLOADER_VERSION=3.4.0

COPY .shared/install-depotdownloader.sh /install.sh
RUN /install.sh "${DEPOTDOWNLOADER_VERSION}" "${TARGETARCH}" /out

# PalServer.sh hard-exits without a steamclient.so alongside the game. That
# file ships with Steam's own SDK redistributables, not the game's depot
# content, so DepotDownloader never produces it - bootstrapping steamcmd
# (+quit, no game install) is the smallest way to get a legitimate copy.
FROM debian:bookworm-slim AS steamclient_installer

RUN <<EOF
set -e
export DEBIAN_FRONTEND=noninteractive
apt -y update
apt -y install ca-certificates curl lib32gcc-s1 lib32stdc++6
rm -rf /var/lib/apt/lists/*
mkdir -p /steamcmd
cd /steamcmd
curl -sqL "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz" | tar zxf -
./steamcmd.sh +quit || true
test -f /steamcmd/linux64/steamclient.so
EOF

FROM debian:bookworm-slim

ARG TARGETARCH

RUN <<EOF
set -e
export DEBIAN_FRONTEND=noninteractive
apt -y update
apt -y install ca-certificates libcurl4 libssl3 squashfs-tools tini
rm -rf /var/lib/apt/lists/*

useradd -m -u 1000 game
mkdir -p /cache /data /game
chown -R game:game /cache /data /game
EOF

COPY --from=depotdownloader_installer /out/DepotDownloader /usr/bin/DepotDownloader
COPY --from=steamclient_installer /steamcmd/linux64/steamclient.so /opt/steamclient.so
COPY .build/palworld_linux_${TARGETARCH} /usr/bin/palworld-mgmt

USER game

EXPOSE 8080/tcp
EXPOSE 8211/udp
EXPOSE 8880/tcp

ENTRYPOINT ["/usr/bin/tini", "--", "palworld-mgmt"]
