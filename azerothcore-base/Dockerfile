FROM ubuntu:24.04 AS ac-builder

ENV AC_ROOT=/azerothcore

COPY scripts/clone-azerothcore.sh /tmp/
RUN /tmp/clone-azerothcore.sh

COPY scripts/clone-module-*.sh /tmp/
RUN <<EOF
find /tmp -name 'clone-module-*.sh' | sort | xargs -I{} bash {}
EOF

COPY scripts/build-azerothcore.sh /tmp/
RUN /tmp/build-azerothcore.sh

FROM debian:bookworm-slim

ENV PATH=/azerothcore/env/dist/bin:$PATH

RUN <<EOF
apt update
apt install -y \
    curl \
    default-mysql-client \
    libarchive-tools \
    libicu72 \
    libmariadb3 \
    libncurses6 \
    libreadline8
rm -rf /var/lib/apt/lists/*

groupadd -g 1000 acore
useradd -u 1000 -g 1000 -m acore

mkdir -p \
    /azerothcore/env/dist/bin \
    /azerothcore/env/dist/etc \
    /azerothcore/env/ref/etc \
    /data \
    /config \
    /logs
EOF

COPY --from=ac-builder /azerothcore/env/dist/bin/authserver  /azerothcore/env/dist/bin/authserver
COPY --from=ac-builder /azerothcore/env/dist/bin/worldserver /azerothcore/env/dist/bin/worldserver
COPY --from=ac-builder /azerothcore/env/dist/bin/dbimport    /azerothcore/env/dist/bin/dbimport
COPY --from=ac-builder /azerothcore/env/dist/etc/            /azerothcore/env/ref/etc/
COPY --from=ac-builder /azerothcore/data/                    /azerothcore/data/
COPY --from=ac-builder /azerothcore/modules/                 /azerothcore/modules/
