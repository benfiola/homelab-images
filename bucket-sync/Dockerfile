FROM debian:bookworm-slim

ARG TARGETARCH

COPY .build/bucket-sync_linux_${TARGETARCH} /usr/bin/bucket-sync

ENTRYPOINT ["bucket-sync"]
