#!/bin/sh
set -e

# renovate: datasource=git-refs depName=https://github.com/SgtBatten/HA_blueprints branch=main
VERSION="075e79a3ac0454aa6c0ff14472fadf5407d33722"
DEST="${ADDONS_DIR}/blueprints/automation"

apk add --no-cache curl
mkdir -p "${DEST}"
curl -o "${DEST}/frigate_camera_notifications.yaml" -fsSL "https://raw.githubusercontent.com/SgtBatten/HA_blueprints/${VERSION}/Frigate_Camera_Notifications/Stable.yaml"
