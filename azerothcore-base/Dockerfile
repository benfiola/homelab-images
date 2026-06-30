FROM ubuntu:24.04 AS ac-builder

ENV AC_ROOT=/azerothcore

COPY clone-azerothcore.sh /tmp/
RUN /tmp/clone-azerothcore.sh

COPY clone-module-*.sh /tmp/
RUN <<EOF
find /tmp -name 'clone-module-*.sh' | sort | xargs -I{} bash {}
EOF

COPY build-azerothcore.sh /tmp/
RUN /tmp/build-azerothcore.sh

FROM debian:bookworm-slim

COPY --from=ac-builder /azerothcore/env/dist/bin/authserver  /azerothcore/env/dist/bin/authserver
COPY --from=ac-builder /azerothcore/env/dist/bin/worldserver /azerothcore/env/dist/bin/worldserver
COPY --from=ac-builder /azerothcore/env/dist/bin/dbimport    /azerothcore/env/dist/bin/dbimport
COPY --from=ac-builder /azerothcore/env/dist/etc/            /azerothcore/env/ref/etc/
COPY --from=ac-builder /azerothcore/data/                    /azerothcore/data/
COPY --from=ac-builder /azerothcore/modules/                 /azerothcore/modules/
