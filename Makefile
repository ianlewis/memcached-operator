REGISTRY := ianmlewis/memcached-operator
BUILD_TAG := dev

GOOS := linux
GOARCH := amd64

build: memcached-operator
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build .

image: build
	docker build -t $(REGISTRY):$(BUILD_TAG) .
