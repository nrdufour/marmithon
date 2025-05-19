.PHONY: build deploy clean

all:
	@echo "Building Marmitton..."
	make build
build:
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	docker buildx build --platform arm64 --tag marmitton .

deploy: build 
	@echo "Deploying Marmitton to Forgejo Registry..."
	docker tag marmitton forge.internal/nemo/marmitton:test
	docker push forge.internal/nemo/marmitton:test

clean:
	@echo "Cleaning up Docker images..."
	docker rmi marmitton