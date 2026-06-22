#!/bin/sh
set -e

VERSION="${1:?version required}"
ARCH="${2:?arch required}"
DEST="${3:?dest required}"

export DEBIAN_FRONTEND=noninteractive
apt -y update
apt -y install curl libc6 libgcc-s1 libgssapi-krb5-2 libicu72 libssl3 libstdc++6 zlib1g
rm -rf /var/lib/apt/lists/*

if [ "${ARCH}" = "amd64" ]; then
    ARCH="x64"
fi

curl -fsSL -o archive.tar.gz "https://builds.dotnet.microsoft.com/dotnet/Sdk/${VERSION}/dotnet-sdk-${VERSION}-linux-${ARCH}.tar.gz"
mkdir -p "${DEST}"
tar -C "${DEST}" -xzf archive.tar.gz
rm archive.tar.gz
