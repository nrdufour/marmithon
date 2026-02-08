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
FROM alpine:3.23
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

# Expose identd port (113) and metrics port (9090)
EXPOSE 113 9090

USER nonroot:nonroot
ENTRYPOINT ["/app/marmithon"]
CMD ["-config", "/app/marmithon.toml"]
