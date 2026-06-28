#!/bin/sh
set -e

VERSION="f003ea11ba96e262199ea130fe8f749605645a52"
DEST="${ADDONS_DIR}/blueprints"

apk add --no-cache curl
mkdir -p "${DEST}"
curl -o "${DEST}/frigate_camera_notifications.yaml" -fsSL "https://raw.githubusercontent.com/SgtBatten/HA_blueprints/${VERSION}/Frigate_Camera_Notifications/Stable.yaml"
