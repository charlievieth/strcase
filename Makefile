MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILE_DIR  := $(dir $(MAKEFILE_PATH))

# Test options
GO             ?= go
GOBIN          ?= $(MAKEFILE_DIR)/bin
GO_COVER_MODE  ?= count
GO_COVER_FLAGS ?= -cover -covermode=$(GO_COVER_MODE)
GO_TEST_FLAGS  ?=
GO_TEST        ?= $(GO) test $(GO_COVER_FLAGS) $(GO_TEST_FLAGS)
RICHGO         ?= $(GOBIN)/richgo
RICHGO_VERSION ?= v0.3.11

# Options for linting comments
COMMENTS       ?= 'TODO|WARN|FIXME|CEV'
GREP           ?= \grep
GREP_COLOR     ?= --color=always
xgrep          := $(GREP) $(GREP_COLOR)

# Arguments for `golangci-lint run`
GOLANGCI               ?= $(GOBIN)/golangci-lint
GOLANGCI_VERSION       ?= v1.52.0
# GOLANGCI_SORT          ?= --sort-results
# GOLANGCI_COLOR         ?= --color=always
# GOLANGCI_SKIP          ?= --skip-dirs='/(gen|phash)($$|/)'
# GOLANGCI_EXTRA_LINTERS ?= --enable=misspell,goimports,gofmt,gocheckcompilerdirectives
# GOLANGCI_EXTRA_FLAGS   ?=
# TODO: enable extra linters like: "misspell", "goimports", "gofmt", "gocheckcompilerdirectives"
# GOLANGCI_FLAGS       ?= $(GOLANGCI_SORT) $(GOLANGCI_COLOR) $(GOLANGCI_SKIP) $(GOLANGCI_EXTRA_LINTERS) $(GOLANGCI_EXTRA_FLAGS)
GOLANGCI_FLAGS         ?= --config=$(MAKEFILE_DIR)/.golangci.yaml

# Benchmark options
NO_TESTS = ^$
# WARN: unused
INDEX_BENCHMARKS = ^BenchmarkIndex('\$'|Hard|Torture|Periodic(Unicode)?)

# Color support.
red = $(shell { tput setaf 1 || tput AF 1; } 2>/dev/null)
yellow = $(shell { tput setaf 3 || tput AF 3; } 2>/dev/null)
cyan = $(shell { tput setaf 6 || tput AF 6; } 2>/dev/null)
term-reset = $(shell { tput sgr0 || tput me; } 2>/dev/null)

.PHONY: all
all: test install

# Install pre-commit hooks and download modules
.PHONY: install
install: pre-commit
	@go mod download
	@go install

# Run verbose tests
testverbose: override GO_TEST_FLAGS += -v

# Run short tests
testshort: override GO_TEST_FLAGS += -short
testshort: override GO_COVER_FLAGS = ''

# Run exhaustive fuzz tests
exhaustive: override GO_TEST_FLAGS += -exhaustive

.PHONY: test testshort testverbose exhaustive
test testshort testverbose exhaustive:
	@$(GO_TEST) ./...

# Assert that there are no skipped tests
.PHONY: testskipped
testskipped:
	@if $(MAKE) testverbose | $(xgrep) --fixed-strings -- '--- SKIP:'; then \
		echo '$(red)FAIL: $(cyan)skipped tests$(term-reset)';               \
		exit 1;                                                             \
	fi

# Test that `go generate` does not change tables.go
.PHONY: testgenerate
testgenerate:
	@$(GO) run -tags gen gen.go -dry-run -skip-tests

# Run all tests (slow)
.PHONY: testall
testall: exhaustive testskipped testgenerate calibrate

bin/richgo:
	@echo '$(yellow)INFO:$(term-reset) Installing richgo version: $(RICHGO_VERSION)'
	@mkdir -p $(GOBIN)
	@GOBIN=$(GOBIN) $(GO) install github.com/kyoh86/richgo@$(RICHGO_VERSION)

# Actual ci target (separate because so that we can override GO)
.PHONY: .ci
.ci: GO = $(RICHGO)
.ci: export RICHGO_FORCE_COLOR=1
.ci: testverbose

# Run and colorize verbose tests for CI
.PHONY: ci
ci: bin/richgo
ci: .ci

# Calibrate brute-force cutover
.PHONY: calibrate
calibrate: GO_COVER_FLAGS =
calibrate: GO_TEST_FLAGS += -v
calibrate:
	@$(GO_TEST) -run Calibrate -calibrate

.PHONY: vet-strcase
vet-strcase:
	@$(GO) vet ./...

.PHONY: vet-gen
vet-gen:
	@$(GO) vet -tags gen gen.go

.PHONY: vet
vet: vet-strcase vet-gen

# Install golangci-lint
bin/golangci-lint:
	@echo '$(yellow)INFO:$(term-reset) Installing golangci-lint version: $(GOLANGCI_VERSION)'
	@mkdir -p $(GOBIN)
	@GOBIN=$(GOBIN) $(GO) install \
		github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)

golangci-lint-gen: override GOLANGCI_EXTRA_FLAGS += --build-tags=gen gen.go
golangci-lint-gen: override GOLANGCI_SKIP =

# Run golangci-lint
.PHONY: golangci-lint-strcase golangci-lint-gen
golangci-lint-strcase golangci-lint-gen: bin/golangci-lint
	@$(GOLANGCI) run $(GOLANGCI_FLAGS)

.PHONY: golangci-lint
golangci-lint: golangci-lint-strcase golangci-lint-gen

.PHONY: lint
lint: vet golangci-lint golangci-lint-gen

# Make sure there aren't any comments that need addressing (TODO or WARN)
#
# NOTE: not currently part of the "lint" target.
.PHONY: lint-comments
lint-comments:
	@if $(xgrep) --line-number --extended-regexp $(COMMENTS) Makefile *.go; then \
		echo '';                                                                 \
		echo '$(red)FAIL: $(cyan)address comments!$(term-reset)';                \
		exit 1;                                                                  \
	fi

# Generate tables.go file
.PHONY: generate
generate:
	@$(GO) generate

# Install pre-commit hook
# TODO: omit on Windows ???
.git/hooks/pre-commit: scripts/pre-commit
	@mkdir -p $(MAKEFILE_DIR)/.git/hooks
	ln -s $(MAKEFILE_DIR)/scripts/pre-commit $(MAKEFILE_DIR)/.git/hooks/pre-commit

# Install pre-commit hooks
# TODO: omit on Windows ???
pre-commit: .git/hooks/pre-commit

.PHONY: clean
clean:
	rm -f cpu.out mem.out
	rm -rf DATA bin
	$(GO) clean -i -cache
