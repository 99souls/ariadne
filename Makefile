# Makefile for ariadne

GO ?= go
PKGS := ./...

.PHONY: all build test race cover lint tidy vet ci snapshot

all: build

build:
	$(GO) build ./...

test:
	$(GO) test $(PKGS)

race:
	$(GO) test -race $(PKGS)

cover:
	$(GO) test -cover -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -func=coverage.out | tail -n 1

lint:
	@[ -x $$(command -v golangci-lint) ] || (echo "golangci-lint not found. Install: https://golangci-lint.run/" && exit 1)
	golangci-lint run ./...

vet:
	$(GO) vet $(PKGS)

tidy:
	$(GO) mod tidy

ci: tidy vet test race

snapshot:
	$(GO) run . -seeds https://example.com -snapshot-interval 5s -checkpoint checkpoint.log | head -n 5

# Count legacy import path occurrences (Wave 2A guard)
legacy-imports:
	@echo "Counting legacy import path occurrences (github.com/99souls/ariadne/engine)" >&2
	@COUNT=$$(grep -R "github.com/99souls/ariadne/engine" -n --include='*.go' . | wc -l | tr -d ' '); \
	echo $$COUNT; \
	if [ -n "$(EXPECTED)" ]; then \
		if [ "$$COUNT" -gt "$(EXPECTED)" ]; then \
			echo "ERROR: Legacy import count ($$COUNT) exceeds EXPECTED=$(EXPECTED)" >&2; exit 1; \
		fi; \
	fi

# Tag a release (usage: make tag VERSION=0.1.0)
tag:
	@if [ -z "$(VERSION)" ]; then echo "VERSION required (e.g., make tag VERSION=0.1.0)"; exit 1; fi
	git tag v$(VERSION)
	git push origin v$(VERSION)
