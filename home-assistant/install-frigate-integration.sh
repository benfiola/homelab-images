#!/bin/sh
set -e

VERSION="${1:?version required}"
DEST="${2:?dest required}"

apk add --no-cache curl
curl -o archive.tar.gz -fsSL "https://github.com/blakeblackshear/frigate-hass-integration/archive/refs/tags/v${VERSION}.tar.gz"
mkdir -p "${DEST}"
tar xzf archive.tar.gz --strip-components=2 -C "${DEST}" "frigate-hass-integration-${VERSION}/custom_components/frigate"
rm archive.tar.gz
