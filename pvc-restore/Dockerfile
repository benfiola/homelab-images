FROM golang:1.25.8 as builder

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/pvc-restore /usr/bin/pvc-restore

ENTRYPOINT ["pvc-restore"]
