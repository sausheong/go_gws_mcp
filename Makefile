BINARY := workspace-mcp
PKG    := ./cmd/workspace-mcp
GOFLAGS ?=

.PHONY: all build run run-http test vet fmt tidy clean help

all: build

build: ## Build the workspace-mcp binary
	go build $(GOFLAGS) -o $(BINARY) $(PKG)

run: build ## Build and run with stdio transport
	./$(BINARY)

run-http: build ## Build and run with streamable-http transport
	./$(BINARY) --transport streamable-http

test: ## Run all tests
	go test ./...

vet: ## go vet
	go vet ./...

fmt: ## gofmt -s -w on the tree
	gofmt -s -w .

tidy: ## go mod tidy
	go mod tidy

clean: ## Remove built binary
	rm -f $(BINARY)

help: ## List targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
