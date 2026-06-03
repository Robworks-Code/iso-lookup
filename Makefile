BINARY  := iso
PKG     := ./cmd/iso
PREFIX  ?= /usr/local
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.DEFAULT_GOAL := help

.PHONY: help build install uninstall test vet fmt tidy clean snapshot release

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary into ./bin
	go build -ldflags '$(LDFLAGS)' -o bin/$(BINARY) $(PKG)

install: ## Install into $(PREFIX)/bin (override with PREFIX=...; may need sudo)
	go build -ldflags '$(LDFLAGS)' -o $(PREFIX)/bin/$(BINARY) $(PKG)

uninstall: ## Remove the installed binary
	rm -f $(PREFIX)/bin/$(BINARY)

test: ## Run the test suite
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format sources
	gofmt -w cmd internal

tidy: ## Tidy go.mod/go.sum
	go mod tidy

clean: ## Remove build artifacts
	rm -rf bin dist

snapshot: ## Build a local goreleaser snapshot (no publish)
	goreleaser release --snapshot --clean

release: ## Run goreleaser against the current tag (CI uses this on tag push)
	goreleaser release --clean
