FROM debian:bookworm-slim

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/pvc-restore /usr/bin/pvc-restore

ENTRYPOINT ["pvc-restore"]
