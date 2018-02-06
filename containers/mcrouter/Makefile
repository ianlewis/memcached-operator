REGISTRY := ianmlewis

MCROUTER_VERSION := 0.36.0
MCROUTER_REV := 0

.PHONY: image test clean mcrouter

mcrouter:
	docker build --build-arg BUILD_DATE=$(shell date --iso-8601=minutes) --build-arg VCS_REF=$(shell git log -1 --oneline | awk '{ print $$1 }') --build-arg MCROUTER_VERSION=v$(MCROUTER_VERSION) --build-arg VERSION=v$(MCROUTER_VERSION)-$(MCROUTER_REV) -t $(REGISTRY)/mcrouter:v$(MCROUTER_VERSION)-$(MCROUTER_REV) .
