# renovate: datasource=github-tags depName=lvmteam/lvm2 versioning=regex:^(?<major>\d+)_(?<minor>\d+)_(?<patch>\d+)$ extractVersion=^v(?<version>\d+_\d+_\d+)$
ARG LVM2_VERSION=2_03_43

FROM debian:trixie-slim@sha256:f6b08b09a05725f9431502d3b5714f5c4576e82c1a84fa1debe0c24a7e0c3ec4 AS builder

ARG LVM2_VERSION

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    build-essential \
    pkg-config \
    libaio-dev \
    libdevmapper-dev \
    libudev-dev \
    tar \
    xfslibs-dev && \
    rm -rf /var/lib/apt/lists/*

COPY build.sh /tmp/build.sh
RUN chmod +x /tmp/build.sh && /tmp/build.sh "${LVM2_VERSION}" /lvm2.tar.gz

FROM debian:trixie-slim@sha256:f6b08b09a05725f9431502d3b5714f5c4576e82c1a84fa1debe0c24a7e0c3ec4

COPY --from=builder /lvm2.tar.gz /lvm2.tar.gz
