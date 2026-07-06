ARG AZEROTHCORE_BASE=ghcr.io/benfiola/homelab-images/azerothcore-base:1.1.0
FROM ${AZEROTHCORE_BASE} AS azerothcore-base

FROM ubuntu:24.04

ARG TARGETARCH

ENV PATH=/azerothcore/env/dist/bin:$PATH

RUN <<EOF
apt update
apt install -y \
    curl \
    default-mysql-client \
    libarchive-tools \
    libicu74 \
    libmysqlclient21 \
    libncurses6 \
    libreadline8 \
    vim
rm -rf /var/lib/apt/lists/*

userdel --remove ubuntu
groupadd -g 1000 acore
useradd -u 1000 -g 1000 -m acore

mkdir -p \
    /azerothcore/env/dist/etc \
    /config \
    /data \
    /logs \
    /tmp/azerothcore

chown acore:acore \
    /azerothcore \
    /azerothcore/env/dist/etc \
    /config \
    /data \
    /logs \
    /tmp/azerothcore
EOF

COPY --from=azerothcore-base /azerothcore/env/dist/bin/ /azerothcore/env/dist/bin/
COPY --from=azerothcore-base /azerothcore/env/ref/etc/  /azerothcore/env/ref/etc/
COPY --from=azerothcore-base /azerothcore/data/         /azerothcore/data/
COPY --from=azerothcore-base /azerothcore/modules/      /azerothcore/modules/

COPY .build/azerothcore_linux_${TARGETARCH} /usr/bin/azerothcore

USER acore

WORKDIR /azerothcore

ENTRYPOINT ["/usr/bin/azerothcore"]
