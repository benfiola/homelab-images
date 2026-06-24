#!/bin/sh
set -e

VERSION="${1:?version required}"
DEST="${2:?dest required}"

apk add --no-cache curl libarchive-tools
curl -o archive.zip -fsSL "https://github.com/dermotduffy/advanced-camera-card/releases/download/v${VERSION}/advanced-camera-card.zip"
mkdir -p "${DEST}"
bsdtar xf archive.zip --strip-components=1 -C "${DEST}"
rm archive.zip
