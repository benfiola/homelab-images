#!/bin/sh
set -e

# renovate: datasource=github-releases depName=xHecktor/light_group_dimmer extractVersion=^v(?<version>.+)$
VERSION="1.1.2"
DEST="${ADDONS_DIR}/custom_components/light_group_dimmer"

apk add --no-cache curl
curl -o archive.tar.gz -fsSL "https://github.com/xHecktor/light_group_dimmer/archive/refs/tags/v${VERSION}.tar.gz"
mkdir -p "${DEST}"
tar xzf archive.tar.gz --strip-components=3 -C "${DEST}" "light_group_dimmer-${VERSION}/custom_components/light_group_dimmer"
rm archive.tar.gz
