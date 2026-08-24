#!/bin/sh
set -e

VERSION="1.7.1"
DEST="${ADDONS_DIR}/custom_components/llmvision"

apk add --no-cache curl
curl -o archive.tar.gz -fsSL "https://github.com/valentinfrlch/ha-llmvision/archive/refs/tags/v${VERSION}.tar.gz"
mkdir -p "${DEST}"
tar xzf archive.tar.gz --strip-components=3 -C "${DEST}" "ha-llmvision-${VERSION}/custom_components/llmvision"
rm archive.tar.gz
