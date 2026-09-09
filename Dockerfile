# syntax=docker/dockerfile:1.7

# --- Builder: cross-compile a static binary on the build platform. ----------
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=0.1.0

WORKDIR /src

# Module downloads are cached across rebuilds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN mkdir -p /out \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
       go build -trimpath -mod=readonly \
         -ldflags "-s -w -X github.com/ofwh/ipa-renamer/src.version=${VERSION}" \
         -o /out/ipa-renamer ./src

# --- Runtime: entrypoint turns IPA_RENAMER_* env vars into CLI arguments. ---
FROM alpine:3.24

ARG VERSION=0.1.0

LABEL org.opencontainers.image.title="ipa-renamer" \
      org.opencontainers.image.description="Scan or watch .ipa files and copy them as <name>@<CFBundleIdentifier>.ipa" \
      org.opencontainers.image.source="https://github.com/ofwh/ipa-renamer" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}"

COPY docker/entrypoint.sh /entrypoint.sh
COPY --from=build /out/ipa-renamer /usr/local/bin/ipa-renamer

RUN chmod +x /entrypoint.sh \
    && mkdir -p /data \
    && chown 65534:65534 /data

WORKDIR /data
ENV HOME=/data

# Non-root (65534 = nobody); /data is owned by it so the built-in defaults
# (input ".", output "renamed") remain writable when no variables are set.
USER 65534:65534

ENTRYPOINT ["/entrypoint.sh"]
