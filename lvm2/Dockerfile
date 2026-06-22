ARG LVM2_VERSION=2.03.38

FROM debian:bookworm-slim AS builder

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

FROM debian:bookworm-slim

COPY --from=builder /lvm2.tar.gz /lvm2.tar.gz
