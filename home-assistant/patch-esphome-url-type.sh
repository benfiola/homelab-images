#!/bin/sh
set -e

HA_DIR="/usr/src/homeassistant"
TRANSLATIONS="${HA_DIR}/homeassistant/components/esphome/translations/en.json"

apk add --no-cache patch jq
patch -p1 -d "${HA_DIR}" < /patches/esphome-url-type.patch

# the image ships generated translations, so strings.json alone isn't enough
jq -s '.[0] * .[1]' "${TRANSLATIONS}" /patches/esphome-url-type-translations.json > /tmp/en.json
mv /tmp/en.json "${TRANSLATIONS}"
apk del patch jq
