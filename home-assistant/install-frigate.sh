#!/bin/sh
set -e

VERSION="5.15.4"
DEST="${ADDONS_DIR}/custom_components/frigate"

apk add --no-cache curl
curl -o archive.tar.gz -fsSL "https://github.com/blakeblackshear/frigate-hass-integration/archive/refs/tags/v${VERSION}.tar.gz"
mkdir -p "${DEST}"
tar xzf archive.tar.gz --strip-components=3 -C "${DEST}" "frigate-hass-integration-${VERSION}/custom_components/frigate"
rm archive.tar.gz
