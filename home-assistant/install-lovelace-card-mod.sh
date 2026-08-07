#!/bin/sh
set -e

VERSION="4.2.1"
DEST="${ADDONS_DIR}/www/lovelace-card-mod"

apk add --no-cache curl
curl -o archive.tar.gz -fsSL "https://github.com/thomasloven/lovelace-card-mod/archive/refs/tags/v${VERSION}.tar.gz"
mkdir -p "${DEST}"
tar xzf archive.tar.gz --strip-components=1 -C "${DEST}" "lovelace-card-mod-${VERSION}/card-mod.js"
rm archive.tar.gz