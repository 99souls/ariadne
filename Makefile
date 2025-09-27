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

api-report:
	@echo "Generating API_REPORT.md" >&2
	$(GO) run ./cmd/apireport -out API_REPORT.md
	@echo "Done" >&2

# Assert zero occurrences of removed legacy path
legacy-imports:
	@echo "Verifying legacy path 'ariadne/packages/engine' is absent" >&2
	@COUNT=$$(grep -R "ariadne/packages/engine" -n --include='*.go' . | wc -l | tr -d ' '); \
	if [ "$$COUNT" -ne 0 ]; then \
		echo "ERROR: Found $$COUNT occurrences of removed legacy path" >&2; \
		grep -R "ariadne/packages/engine" -n --include='*.go' . | head -n 20 >&2; \
		exit 1; \
	fi; \
	echo "OK (0)" >&2

# Tag a release (usage: make tag VERSION=0.1.0)
tag:
	@if [ -z "$(VERSION)" ]; then echo "VERSION required (e.g., make tag VERSION=0.1.0)"; exit 1; fi
	git tag v$(VERSION)
	git push origin v$(VERSION)
