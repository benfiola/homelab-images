#!/bin/sh
set -e

source="source:$SOURCE_BUCKET"
if [ -n "${SOURCE_CRYPT_PASSWORD_PLAINTEXT:-}" ]; then
  export RCLONE_CONFIG_SOURCECRYPT_TYPE=crypt
  export RCLONE_CONFIG_SOURCECRYPT_REMOTE="$source"
  export RCLONE_CONFIG_SOURCECRYPT_PASSWORD="$(rclone obscure "$SOURCE_CRYPT_PASSWORD_PLAINTEXT")"
  if [ -n "${SOURCE_CRYPT_PASSWORD2_PLAINTEXT:-}" ]; then
    export RCLONE_CONFIG_SOURCECRYPT_PASSWORD2="$(rclone obscure "$SOURCE_CRYPT_PASSWORD2_PLAINTEXT")"
  fi
  source="sourcecrypt:"
fi

destination="destination:$DESTINATION_BUCKET"
if [ -n "${DESTINATION_CRYPT_PASSWORD_PLAINTEXT:-}" ]; then
  export RCLONE_CONFIG_DESTINATIONCRYPT_TYPE=crypt
  export RCLONE_CONFIG_DESTINATIONCRYPT_REMOTE="$destination"
  export RCLONE_CONFIG_DESTINATIONCRYPT_PASSWORD="$(rclone obscure "$DESTINATION_CRYPT_PASSWORD_PLAINTEXT")"
  if [ -n "${DESTINATION_CRYPT_PASSWORD2_PLAINTEXT:-}" ]; then
    export RCLONE_CONFIG_DESTINATIONCRYPT_PASSWORD2="$(rclone obscure "$DESTINATION_CRYPT_PASSWORD2_PLAINTEXT")"
  fi
  destination="destinationcrypt:"
fi

exec rclone sync "$source" "$destination"
