REGISTRY := ianmlewis/memcached-operator
BUILD_TAG := dev
VERSION := $(shell cat VERSION)

GOOS := linux
GOARCH := amd64

.PHONY: image test clean

memcached-operator:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build .

image: build
	docker build --build-arg BUILD_DATE=$(shell date --iso-8601=minutes) --build-arg VCS_REF=$(shell git log -1 --oneline | awk '{ print $$1 }') --build-arg VERSION=$(VERSION) -t $(REGISTRY):v$(VERSION) .

test:
	go test -v ./...

clean:
	rm -f memcached-operator
