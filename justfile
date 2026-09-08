bin     := "bin/marmithon"
commit  := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
built   := `date -u +%Y-%m-%dT%H:%M:%SZ`
ldflags := "-s -w -X marmithon/command.GitCommit=" + commit + " -X marmithon/command.BuildTime=" + built
image   := "marmithon"

# Built and shipped with cgo off, so everything here runs that way too.
export CGO_ENABLED := "0"

_default:
    @just --list --unsorted

# Compile the binary into bin/.
build:
    go build -trimpath -ldflags "{{ ldflags }}" -o {{ bin }} .

# Run from source with the dev config.
run:
    go run . -config dev.toml

# Run the tests.
test:
    go test ./...

# vet and staticcheck. Fast enough to run on every save.
lint:
    go vet ./...
    staticcheck ./...

# Format every tracked Go and Nix file in place.
fmt:
    gofmt -w $(git ls-files '*.go')
    nixfmt $(git ls-files '*.nix')

# Fail if any tracked Go or Nix file needs formatting.
fmt-check:
    #!/usr/bin/env bash
    unformatted=$(gofmt -l $(git ls-files '*.go'))
    if [ -n "$unformatted" ]; then
        echo "these files need gofmt:"; echo "$unformatted"; exit 1
    fi
    nixfmt --check $(git ls-files '*.nix')

# What CI runs, so a red build is reproducible in one command.
check: fmt-check lint test confusables

# Characters that render like ASCII, or render as nothing at all, but compare
# unequal. CI runs this in its own workflow, without a path filter, so it
# covers prose too.

# Grep for characters confusable with ASCII, or invisible.
confusables:
    #!/usr/bin/env bash
    pattern='[\x{2010}-\x{2015}\x{2212}\x{2018}\x{2019}\x{201A}\x{201C}\x{201D}\x{201E}\x{2032}\x{2033}\x{00B7}\x{2026}\x{00D7}\x{00A0}\x{200B}-\x{200D}\x{2060}\x{FEFF}]'
    if hits=$(git ls-files | xargs grep -nIP "$pattern" 2>/dev/null); then
        echo "characters confusable with ASCII, or invisible:"
        echo "$hits"
        echo
        echo "Use - . ... x and a plain space instead."
        exit 1
    fi

# Build the image for the local architecture. CI builds both arches natively.
docker:
    docker build \
      --build-arg IMAGE_REVISION={{ commit }} \
      --build-arg IMAGE_CREATED={{ built }} \
      --tag {{ image }} .

# Push a locally built image to forge.internal as :test.
deploy: docker
    docker tag {{ image }} forge.internal/nemo/{{ image }}:test
    docker push forge.internal/nemo/{{ image }}:test

# Remove build output.
clean:
    rm -rf bin/
