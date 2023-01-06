MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILE_DIR  := $(dir $(MAKEFILE_PATH))

# Test options
GO             ?= go
GO_COVER_MODE  ?= count
GO_COVER_FLAGS ?= -cover -covermode=$(GO_COVER_MODE)
GO_TEST_FLAGS  ?=
GO_TEST        ?= $(GO) test $(GO_COVER_FLAGS) $(GO_TEST_FLAGS)

# Options for linting comments
COMMENTS       ?= 'TODO|WARN|FIXME|CEV'
GREP           ?= \grep
GREP_COLOR     ?= --color=always
xgrep          := $(GREP) $(GREP_COLOR)

# Arguments for `golangci-lint run`
GOLANGCI             ?= $(MAKEFILE_DIR)/bin/golangci-lint
GOLANGCI_VERSION     ?= v1.50.1
GOLANGCI_SORT        ?= --sort-results
GOLANGCI_COLOR       ?= --color=always
GOLANGCI_SKIP        ?= --skip-dirs='internal/(gen|ucd)'
GOLANGCI_EXTRA_FLAGS ?=
GOLANGCI_FLAGS       ?= $(GOLANGCI_SORT) $(GOLANGCI_COLOR) $(GOLANGCI_SKIP) $(GOLANGCI_EXTRA_FLAGS)

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
all: test

# Run verbose tests
testverbose: override GO_TEST_FLAGS += -v

# Run short tests
testshort: override GO_TEST_FLAGS += -short
testshort: override GO_COVER_FLAGS = ''

# Run exhaustive fuzz tests
exhaustive: override GO_TEST_FLAGS += -exhaustive

.PHONY: test testshort testverbose exhaustive
test testshort testverbose exhaustive:
	@$(GO_TEST)

# Assert that there are no skipped tests
.PHONY: skipped_tests
skipped_tests:
	@if $(MAKE) testverbose | $(xgrep) --fixed-strings -- '--- SKIP:'; then \
		echo '$(red)FAIL: $(cyan)skipped tests$(term-reset)';               \
		exit 1;                                                             \
	fi

# Test that `go generate` does not change tables.go
.PHONY: test_generate
test_generate:
	@$(GO) run -tags gen gen.go -dry-run -skip-tests

# Run all tests (slow)
.PHONY: test-all
test-all: test exhaustive skipped_tests test_generate

# Calibrate brute-force cutover
.PHONY: calibrate
calibrate:
	@$(GO_TEST) -run TestCalibrate -calibrate

.PHONY: vet-strcase
vet-strcase:
	@$(GO) vet

.PHONY: vet-gen
vet-gen:
	@$(GO) vet -tags gen gen.go

.PHONY: vet
vet: vet-strcase vet-gen

# Install golangci-lint
bin/golangci-lint:
	@echo '$(yellow)INFO:$(term-reset) Installing golangci-lint version: $(GOLANGCI_VERSION)'
	@mkdir -p $(MAKEFILE_DIR)/bin
	@GOBIN=$(MAKEFILE_DIR)/bin $(GO) install \
		github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)

golangci-lint-gen: override GOLANGCI_EXTRA_FLAGS += --build-tags=gen gen.go
golangci-lint-gen: override GOLANGCI_SKIP =

.PHONY: golangci-lint golangci-lint-gen
golangci-lint golangci-lint-gen: bin/golangci-lint
	@$(GOLANGCI) run $(GOLANGCI_FLAGS)

.PHONY: lint_comments
lint_comments:
	@if $(xgrep) --line-number --extended-regexp $(COMMENTS) Makefile *.go; then \
		echo '';                                                                 \
		echo '$(red)FAIL: $(cyan)address comments!$(term-reset)';                \
		exit 1;                                                                  \
	fi

.PHONY: lint
lint: vet golangci-lint golangci-lint-gen

.PHONY: clean
clean:
	rm -f cpu.out mem.out
	rm -rf DATA bin
	$(GO) clean -i -cache
