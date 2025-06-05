FROM docker.io/library/golang:1.24.3-alpine3.21 as build

WORKDIR /marmithon

## git is needed for the go build process with the library github.com/earthboundkid/versioninfo
RUN apk update && apk add --no-cache \
    tini-static \
    git
COPY . .
RUN go build -ldflags="-s -w"

# -----------------------------------------------------------------------------
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot

COPY --from=build /marmithon/marmithon /app/marmithon
COPY --from=build --chown=nonroot:nonroot /sbin/tini-static /sbin/tini

ENTRYPOINT ["/sbin/tini", "--", "/app/marmithon"]

LABEL "org.opencontainers.image.title"="marmithon"