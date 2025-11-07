FROM docker.io/library/golang:1.25.4-alpine3.21 AS build

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
FROM docker.io/library/alpine:3.22

# Install required packages including Tailscale
RUN apk update && apk add --no-cache \
    ca-certificates \
    iptables \
    ip6tables \
    iproute2 \
    curl \
    su-exec \
    tailscale

# Create a non-root user
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot

# Copy the binary and config from build stage
COPY --from=build /marmithon/marmithon /app/marmithon
COPY --from=build /marmithon/marmithon.toml /app

# Copy entrypoint script
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Tailscale needs to run as root to set up networking
# But marmithon will run as nonroot user in the entrypoint script

WORKDIR /app

# Environment variables for Tailscale configuration
ENV TS_AUTHKEY=""
ENV EXIT_NODE_IP=""

ENTRYPOINT ["/app/entrypoint.sh"]
