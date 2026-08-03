#!/bin/sh
set -e

VERSION="${1:?version required}"
ARCH="${2:?arch required}"
DEST="${3:?dest required}"

export DEBIAN_FRONTEND=noninteractive

apt -y update
apt -y install ca-certificates curl libssl3 unzip
rm -rf /var/lib/apt/lists/*

if [ "${ARCH}" = "amd64" ]; then
    ARCH="x64"
fi
curl -fsSL -o archive.zip "https://github.com/SteamRE/DepotDownloader/releases/download/DepotDownloader_${VERSION}/DepotDownloader-linux-${ARCH}.zip"
mkdir -p extract
unzip -d extract archive.zip
mkdir -p "${DEST}"
mv extract/DepotDownloader "${DEST}"
rm -rf extract archive.zip
