#!/bin/sh
set -e

# renovate: datasource=github-releases depName=lovelylain/hass_ingress
VERSION="1.4.0"
DEST="${ADDONS_DIR}/custom_components/ingress"

apk add --no-cache curl libarchive-tools
curl -o archive.zip -fsSL "https://github.com/lovelylain/hass_ingress/releases/download/${VERSION}/ingress.zip"
mkdir -p "${DEST}"
bsdtar xf archive.zip -C "${DEST}"
rm archive.zip
