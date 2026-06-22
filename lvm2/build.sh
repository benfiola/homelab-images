#!/bin/sh
# Builds LVM2
# Should be run in a debian:bookworm-slim image
# Requires the following (debian) packages: curl build-essential pkg-config libaio-dev libdevmapper-dev libudev-dev tar xfslibs-dev
set -e

VERSION="${1}"
OUTPUT_ARCHIVE="${2}"

if [ -z "${VERSION}" ] || [ -z "${OUTPUT_ARCHIVE}" ]; then
    echo "Usage: ${0} <VERSION> <OUTPUT_ARCHIVE>"
    exit 1
fi

OUTPUT_ARCHIVE="$(readlink -f "${OUTPUT_ARCHIVE}")"

TEMP_DIR="$(mktemp -d)"
trap "rm -rf '$TEMP_DIR'" EXIT

curl -fsSL -o "${TEMP_DIR}/archive.tar.gz" "https://sourceware.org/pub/lvm2/releases/LVM2.${VERSION}.tgz"
mkdir -p "${TEMP_DIR}/source"
tar xvzf "${TEMP_DIR}/archive.tar.gz" -C "${TEMP_DIR}/source" --strip-components=1
rm -f "${TEMP_DIR}/archive.tar.gz"

mkdir -p "${TEMP_DIR}/build"
cd "${TEMP_DIR}/build"
"${TEMP_DIR}/source/configure" \
  --prefix=/usr \
  --sysconfdir=/etc \
  --sbindir=/sbin \
  --localstatedir=/var \
  --enable-static_link \
  --disable-dependency-tracking \
  --with-thin=internal
make

mkdir -p "${TEMP_DIR}/package"
DESTDIR="${TEMP_DIR}/package" make install

cd "${TEMP_DIR}/package"
tar cvzf "${OUTPUT_ARCHIVE}" .
