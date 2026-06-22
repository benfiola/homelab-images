FROM golang:1.25.8 as builder

ARG TARGETARCH
ARG TARGETPLATFORM

COPY $TARGETPLATFORM/router-policy-sync /usr/bin/router-policy-sync

ENTRYPOINT ["router-policy-sync"]
