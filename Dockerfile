# syntax=docker/dockerfile:1.7

# --- Builder: cross-compile a static binary on the build platform. ----------
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION

# Required: -V/--version reports this, so the build fails when it is unset.
RUN if [ -z "${VERSION}" ]; then \
      echo "error: the VERSION build arg is required, e.g. --build-arg VERSION=0.1.0." >&2; \
      exit 1; \
    fi

WORKDIR /src

# Module downloads are cached across rebuilds.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN mkdir -p /out \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
       go build -trimpath -mod=readonly \
         -ldflags "-s -w -X main.version=${VERSION}" \
         -o /out/ipa-renamer ./src

# --- Runtime: everything lives under /app. ----------------------------------
FROM alpine:latest

ARG VERSION

LABEL org.opencontainers.image.title="ipa-renamer" \
      org.opencontainers.image.description="Rename ipa file with bundle identifier read from the app's Info.plist inside the archive." \
      org.opencontainers.image.source="https://github.com/ofwh/ipa-renamer" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION}"

COPY docker/entrypoint.sh /app/entrypoint.sh
COPY --from=build /out/ipa-renamer /app/ipa-renamer

RUN chmod +x /app/entrypoint.sh \
    && mkdir -p /app/in /app/out \
    && chown -R 65534:65534 /app

WORKDIR /app

ENV HOME=/app \
    IDLE_TIMEOUT=5 \
    RECURSIVE=1 \
    WATCH=1

# Non-root (65534 = nobody), owning the directories above.
USER 65534:65534

ENTRYPOINT ["/app/entrypoint.sh"]
