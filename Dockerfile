FROM docker.io/library/golang:1.25.1-alpine3.21 as build

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
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot

COPY --from=build /marmithon/marmithon /app/marmithon
COPY --from=build /marmithon/marmithon.toml /app

CMD ["/app/marmithon", "-config", "/app/marmithon.toml"]
