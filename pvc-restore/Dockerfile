FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/pvc-restore_linux_${TARGETARCH} /usr/bin/pvc-restore

ENTRYPOINT ["pvc-restore"]
