# Makefile for ariadne

GO ?= go

# Explicit module directories (root has no go.mod)
MODULES := engine cli tools/apireport
ENGINE_MOD := engine
CLI_MOD := cli
TOOLS_APIREPORT := tools/apireport

.PHONY: all build test race cover lint tidy vet ci snapshot

all: build

build:
	@for m in $(MODULES); do echo "==> build $$m"; (cd $$m && $(GO) build ./... ) || exit 1; done

test:
	@for m in $(MODULES); do echo "==> test $$m"; (cd $$m && $(GO) test ./... ) || exit 1; done

race:
	@for m in $(MODULES); do echo "==> race $$m"; (cd $$m && $(GO) test -race ./... ) || exit 1; done

cover:
	@rm -f coverage.out
	@first=1; \
	for m in $(MODULES); do \
		echo "==> cover $$m"; \
		(cd $$m && $(GO) test -covermode=atomic -coverprofile=coverage.tmp ./... ) || exit 1; \
		if [ $$first -eq 1 ]; then \
			cat $$m/coverage.tmp > coverage.out; \
			first=0; \
		else \
			tail -n +2 $$m/coverage.tmp >> coverage.out; \
		fi; \
		rm $$m/coverage.tmp; \
	done; \
	$(GO) tool cover -func=coverage.out | tail -n 1

lint:
	@[ -x $$(command -v golangci-lint) ] || (echo "golangci-lint not found. Install: https://golangci-lint.run/" && exit 1)
	@status=0; \
	for m in $(MODULES); do \
	  echo "==> lint $$m"; \
	  (cd $$m && golangci-lint run ./... ) || { status=1; break; }; \
	done; \
	exit $$status

vet:
	@for m in $(MODULES); do echo "==> vet $$m"; (cd $$m && $(GO) vet ./... ) || exit 1; done

tidy:
	@for m in $(MODULES); do echo "==> tidy $$m"; (cd $$m && $(GO) mod tidy ) || exit 1; done

ci: tidy vet test race

snapshot:
	cd $(CLI_MOD) && $(GO) run ./cmd/ariadne -seeds https://example.com -snapshot-interval 5s -checkpoint checkpoint.log | head -n 5

api-report:
	@echo "Generating API_REPORT.md" >&2
	$(GO) run ./tools/apireport -out API_REPORT.md
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
