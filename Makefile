GO ?= go
PKGS := ./...

BIN_DIR ?= bin
BINARY_NAME ?= swiftseer
GOOS := $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
CGO_ENABLED ?= 0
EXT :=
ifeq ($(GOOS),windows)
EXT := .exe
endif
OUT := $(BIN_DIR)/$(BINARY_NAME)$(EXT)

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/deichbewohner/swiftseer/internal/version.Version=$(VERSION) \
           -X github.com/deichbewohner/swiftseer/internal/version.GitCommit=$(COMMIT) \
           -X github.com/deichbewohner/swiftseer/internal/version.BuildDate=$(DATE)

# Enable race detector by default on non-Windows
RACE ?=
ifneq ($(GOOS),windows)
RACE ?= -race
endif
TEST_FLAGS ?= $(RACE) -count=1

.PHONY: all build build-all test clean tidy run cover help ci fmt lint vet hooks

all: build

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	@echo "Building $(OUT) version $(VERSION)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
	  $(GO) build -ldflags "$(LDFLAGS)" -o $(OUT) .

OSARCHES ?= linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/arm64
build-all: $(BIN_DIR)
	@set -e; for oa in $(OSARCHES); do \
	  target_os=$${oa%/*}; target_arch=$${oa#*/}; \
	  ext=""; [ "$$target_os" = "windows" ] && ext=".exe"; \
	  out="$(BIN_DIR)/$(BINARY_NAME)-$$target_os-$$target_arch$$ext"; \
	  echo ">> Building $$out"; \
	  GOOS=$$target_os GOARCH=$$target_arch CGO_ENABLED=$(CGO_ENABLED) \
	    $(GO) build -ldflags "$(LDFLAGS)" -o $$out .; \
	done

test:
	@echo "Running tests..."
	$(GO) test $(PKGS) $(TEST_FLAGS)

cover:
	@echo "Running tests with coverage..."
	$(GO) test $(PKGS) -coverprofile=coverage.out
	@$(GO) tool cover -func=coverage.out | tail -n 1

tidy:
	$(GO) mod tidy

run:
	$(GO) run .

fmt:
	@$(GO) fmt ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR) $(GOCACHE) $(GOMODCACHE) coverage.out

ci: test build

help:
	@echo "Targets:"
	@echo "  build       Build binary to $(OUT)"
	@echo "  build-all   Cross-compile for all targets ($(OSARCHES))"
	@echo "  test        Run unit tests ($(TEST_FLAGS))"
	@echo "  cover       Run tests with coverage summary"
	@echo "  run         Run the CLI with go run"
	@echo "  tidy        Run go mod tidy"
	@echo "  fmt         Format code with go fmt"
	@echo "  lint        Run golangci-lint (requires it to be installed)"
	@echo "  clean       Remove build artifacts and caches"
	@echo "  ci          Run tests and build"
	@echo "  hooks       Install Git hooks (fmt on commit)"

hooks:
	@git config core.hooksPath .githooks
	@chmod +x .githooks/* 2>/dev/null || true
	@echo "Git hooks installed (core.hooksPath=.githooks)"

GOLANGCI_LINT ?= golangci-lint
GOLANGCI_LINT_FLAGS ?=

lint:
	@if ! command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		echo "golangci-lint not found. Install from https://golangci-lint.run or 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'"; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS)

vet:
	$(GO) vet ./...
