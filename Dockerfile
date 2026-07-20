# -----------------------------------------------------------------------------
# OCI image labels (set via --build-arg; defaults apply when omitted)
# IMAGE_REVISION and IMAGE_CREATED should be passed at build time
ARG IMAGE_SOURCE=https://forge.internal/nemo/marmithon
ARG IMAGE_REVISION=unknown
ARG IMAGE_CREATED=unknown
ARG IMAGE_VERSION=0.1.0
ARG IMAGE_TITLE=Marmithon
ARG IMAGE_DESCRIPTION=Simple IRC bot for the #souk channel
ARG IMAGE_AUTHORS=nemo
ARG IMAGE_LICENSES=MIT
ARG IMAGE_VENDOR=ptinem
ARG IMAGE_URL=https://forge.internal/nemo/marmithon
ARG IMAGE_DOCUMENTATION=https://forge.internal/nemo/marmithon
# -----------------------------------------------------------------------------

FROM docker.io/library/golang:1.25.5-alpine3.21 AS build

ARG TARGETOS=linux
ARG TARGETARCH=arm64

WORKDIR /marmithon

## git is needed to get commit info for version
RUN apk update && apk add --no-cache \
    git
COPY . .
RUN COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    BUILD_TIME=$(date -u +"%Y-%m-%d %H:%M:%S UTC") && \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w -X 'marmithon/command.GitCommit=${COMMIT}' -X 'marmithon/command.BuildTime=${BUILD_TIME}'"

# -----------------------------------------------------------------------------
FROM alpine:3.24

# Re-declare ARGs so they're available in this stage
ARG IMAGE_SOURCE
ARG IMAGE_REVISION
ARG IMAGE_CREATED
ARG IMAGE_VERSION
ARG IMAGE_TITLE
ARG IMAGE_DESCRIPTION
ARG IMAGE_AUTHORS
ARG IMAGE_LICENSES
ARG IMAGE_VENDOR
ARG IMAGE_URL
ARG IMAGE_DOCUMENTATION

# Install useful utilities for debugging and management
RUN apk add --no-cache \
    wget \
    curl \
    ca-certificates \
    bind-tools \
    busybox-extras \
    netcat-openbsd

# Create nonroot user
RUN addgroup -g 65532 nonroot && \
    adduser -D -u 65532 -G nonroot nonroot

COPY --from=build /marmithon/marmithon /app/marmithon
COPY --from=build /marmithon/marmithon.toml /app

# OCI annotations (see https://specs.opencontainers.org/image-spec/annotations/)
LABEL org.opencontainers.image.source=$IMAGE_SOURCE
LABEL org.opencontainers.image.revision=$IMAGE_REVISION
LABEL org.opencontainers.image.created=$IMAGE_CREATED
LABEL org.opencontainers.image.version=$IMAGE_VERSION
LABEL org.opencontainers.image.title=$IMAGE_TITLE
LABEL org.opencontainers.image.description=$IMAGE_DESCRIPTION
LABEL org.opencontainers.image.authors=$IMAGE_AUTHORS
LABEL org.opencontainers.image.licenses=$IMAGE_LICENSES
LABEL org.opencontainers.image.vendor=$IMAGE_VENDOR
LABEL org.opencontainers.image.url=$IMAGE_URL
LABEL org.opencontainers.image.documentation=$IMAGE_DOCUMENTATION

# Expose identd port (113) and metrics port (9090)
EXPOSE 113 9090

USER nonroot:nonroot
ENTRYPOINT ["/app/marmithon"]
CMD ["-config", "/app/marmithon.toml"]
