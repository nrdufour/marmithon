.PHONY: build build-local deploy clean

all:
	@echo "Building Marmithon..."
	make build

build-local:
	@echo "Building Marmithon locally..."
	@COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILD_TIME=$$(date -u); \
	go build -ldflags="-X 'marmithon/command.GitCommit=$${COMMIT}' -X 'marmithon/command.BuildTime=$${BUILD_TIME}'"

build:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	docker buildx build --platform arm64 --tag marmithon .

deploy: build 
	@echo "Deploying Marmithon to Forgejo Registry..."
	docker tag marmithon forge.internal/nemo/marmithon:test
	docker push forge.internal/nemo/marmithon:test

clean:
	@echo "Cleaning up Docker images..."
	docker rmi marmithon