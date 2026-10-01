# Common tasks. Run `make help` to list them.
NAME      := portbell
PKG       := github.com/tejjasdev/portbell
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE      ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -s -w -X $(PKG)/internal/version.Version=$(VERSION) \
             -X $(PKG)/internal/version.Commit=$(COMMIT) \
             -X $(PKG)/internal/version.Date=$(DATE)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

.PHONY: build
build: ## Build ./bin/portbell for this machine
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(NAME) ./cmd/$(NAME)

.PHONY: run
run: ## Run from source on the default port
	go run ./cmd/$(NAME)

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: race
race: ## Run all tests with the race detector (Linux and macOS)
	go test -race ./...

.PHONY: lint
lint: ## Run golangci-lint (install it first)
	golangci-lint run

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

.PHONY: tidy
tidy: ## Update go.mod and go.sum
	go mod tidy

.PHONY: cross
cross: ## Check the program builds for Windows, macOS and Linux
	GOOS=linux   GOARCH=amd64 go build -o /dev/null ./cmd/$(NAME)
	GOOS=darwin  GOARCH=arm64 go build -o /dev/null ./cmd/$(NAME)
	GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/$(NAME)

.PHONY: check
check: fmt-check vet test ## What CI runs, minus the linter

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "These files need gofmt:"; gofmt -l .; exit 1)

.PHONY: vet
vet:
	go vet ./...

.PHONY: clean
clean: ## Remove build output
	rm -rf bin dist
