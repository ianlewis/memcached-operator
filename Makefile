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
	$(eval NOHEADERS = $(shell find . -path ./vendor -prune -o -path ./third_party -prune -o -type f \( -iname \*.go -o -iname \*.yaml -o -iname \*.yml -o -iname \*.sh \) -print0 | xargs -r -0 grep -Le "Copyright [0-9][0-9][0-9][0-9] Google LLC"))
	@true
	@if [ "$(NOHEADERS)" != "" ]; then \
		echo "License headers missing for files:"; \
		echo $(NOHEADERS); \
		false; \
	fi
	go test -v ./...

clean:
	rm -f memcached-operator
