ARG LVM2_VERSION=2.03.38

FROM debian:bookworm-slim AS builder

ARG LVM2_VERSION

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
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
RUN chmod +x /tmp/build.sh && /tmp/build.sh "${LVM2_VERSION}" /tmp/lvm2.tar.gz

FROM debian:bookworm-slim

COPY --from=builder /tmp/lvm2.tar.gz /tmp/lvm2.tar.gz
RUN tar xzf /tmp/lvm2.tar.gz -C / && rm /tmp/lvm2.tar.gz

ENTRYPOINT ["/sbin/lvm"]
