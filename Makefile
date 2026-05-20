.DEFAULT_GOAL := help

BIN_DIR := bin
BINARY := $(BIN_DIR)/overseer

.PHONY: build test test-integration update-golden mocks lint run clean tidy qa-tmux help

build: ## Build the overseer binary
	mkdir -p $(BIN_DIR)
	go build -o $(BINARY) ./cmd/overseer/

test: ## Run the unit test suite
	go test -race -cover ./...

test-integration: ## Run integration tests
	go test -race -tags=integration ./...

update-golden: ## Update golden test files
	go test -update ./...

mocks: ## Regenerate testify mocks from domain ports (see .mockery.yml)
	@command -v mockery >/dev/null 2>&1 || { echo "mockery not found in PATH; install with: go install github.com/vektra/mockery/v3@latest"; exit 1; }
	go generate ./...

lint: ## Run static analysis
	golangci-lint run ./...

run: ## Build and run overseer
	$(MAKE) build
	./$(BINARY)

clean: ## Remove build artifacts and test cache
	rm -rf bin/ coverage.* && go clean -testcache

tidy: ## Tidy module dependencies
	go mod tidy

.PHONY: qa-tmux
qa-tmux: build  ## Run tmux-based end-to-end QA scenarios
	@mkdir -p .sisyphus/evidence
	@bash scripts/qa-tmux.sh

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
