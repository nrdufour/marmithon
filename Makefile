
.PHONY: build

all: build

multiarch:
	sudo podman run --rm --privileged docker.io/multiarch/qemu-user-static --reset -p yes

build:
	podman build --arch arm --override-arch arm -t marmitton .

deploy:
	podman tag marmitton registry.home.arpa:5000/marmitton
	podman push registry.home.arpa:5000/marmitton

