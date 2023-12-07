FROM docker.io/library/golang:1.21.5-alpine3.18 as build

WORKDIR /marmitton

RUN apk update && apk add --no-cache tini-static
COPY . .
RUN go build -ldflags="-s -w"

# -----------------------------------------------------------------------------
FROM gcr.io/distroless/static:nonroot
USER nonroot:nonroot

COPY --from=build /marmitton/marmitton /app/marmitton
COPY --from=build --chown=nonroot:nonroot /sbin/tini-static /sbin/tini

ENTRYPOINT ["/sbin/tini", "--", "/app/marmitton"]
