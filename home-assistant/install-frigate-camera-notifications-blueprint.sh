#!/bin/sh
set -e

# renovate: datasource=git-refs depName=https://github.com/SgtBatten/HA_blueprints branch=main
VERSION="4078a909511b707efdcef698735e37c9a4d119d9"
DEST="${ADDONS_DIR}/blueprints/automation"

apk add --no-cache curl
mkdir -p "${DEST}"
curl -o "${DEST}/frigate_camera_notifications.yaml" -fsSL "https://raw.githubusercontent.com/SgtBatten/HA_blueprints/${VERSION}/Frigate_Camera_Notifications/Stable.yaml"
