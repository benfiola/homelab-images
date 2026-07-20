ARG SEERR_VERSION=v3.3.0
ARG TARGETPLATFORM

FROM node:22.22.2-alpine3.23@sha256:8ea2348b068a9544dae7317b4f3aafcdc032df1647bb7d768a05a5cad1a7683f AS builder
ARG SEERR_VERSION
ARG TARGETPLATFORM
ENV TARGETPLATFORM=${TARGETPLATFORM:-linux/amd64}
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"

RUN corepack enable

WORKDIR /app

RUN <<EOF
apk add git
git clone https://github.com/seerr-team/seerr.git .
git checkout ${SEERR_VERSION}
EOF

COPY *.patch .
RUN <<EOF
set -e

git config user.email "builder@localhost"
git config user.name "Builder"
for patch in *.patch; do
  if [ -f "$patch" ]; then
    echo "Applying patch: $patch"
    git apply "$patch"
  fi
done
git add -A
git commit -m "Apply custom patches"
git rev-parse HEAD > /app/.commit-hash
EOF

RUN <<EOF
case "${TARGETPLATFORM}" in
'linux/arm64' | 'linux/arm/v7')
  apk update && \
  apk add --no-cache python3 make g++ gcc libc6-compat bash && \
  npm install --global node-gyp
  ;;
esac
EOF

RUN --mount=type=cache,id=pnpm,target=/pnpm/store <<EOF
export COMMIT_TAG=$(cat /app/.commit-hash)
CYPRESS_INSTALL_BINARY=0 pnpm install --frozen-lockfile
pnpm build
rm -rf .next/cache
EOF

FROM seerr/seerr:${SEERR_VERSION}

USER node:node

COPY --chown=node:node --from=builder /app/.next /app/.next
COPY --chown=node:node --from=builder /app/dist /app/dist
COPY --chown=node:node --from=builder /app/.commit-hash /tmp/.commit-hash

RUN <<EOF
set -e

echo "{\"commitTag\": \"$(cat /tmp/.commit-hash)\"}" > /app/committag.json 
rm /tmp/.commit-hash
EOF
