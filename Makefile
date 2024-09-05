# vim: ts=4 sw=4

# Packages to run exhaustive tests against
EXHAUSTIVE_PKGS = github.com/charlievieth/strcase \
	github.com/charlievieth/strcase/bytcase

# Run tests and linters. If this passes then CI tests
# should also pass.
.PHONY: all
all: install test testbenchmarks testgenerate testgenpkg vet golangci-lint

include common.mk

# Install pre-commit hooks and download modules
.PHONY: install
install: pre-commit
	@$(GO) mod download
	@$(GO) install

# Run verbose tests
testverbose: override GO_TEST_FLAGS += -v

# Run short tests
testshort: override GO_TEST_FLAGS += -short
testshort: override GO_COVER_FLAGS = ''

# Fuzz test with invalid runes
testinvalid: override GO_TEST_FLAGS += -invalid
testinvalid: override GO_TEST_FLAGS += -run 'Test\w+Fuzz'

.PHONY: test testshort testverbose testinvalid
test testshort testverbose testinvalid:
	@GOGC=$(GO_GOGC) $(GO_TEST) ./...

# Run exhaustive fuzz tests
.PHONY: exhaustive
exhaustive:
	@GOGC=$(GO_GOGC) $(GO_TEST) $(EXHAUSTIVE_PKGS) -exhaustive

# Assert that there are no skipped tests
.PHONY: testskipped
testskipped:
	@if $(MAKE) testverbose | $(xgrep) --fixed-strings -- '--- SKIP:'; then \
		echo '';                                                            \
		echo '$(red)FAIL: $(cyan)^^^ skipped tests ^^^$(term-reset)';       \
		echo '';                                                            \
		exit 1;                                                             \
	fi

# The gen package is separate from the strcase package (so we don't pollute
# our go.mod with its dependencies) so we need to cd into its directory to
# run the tests.
.PHONY: testgenpkg
testgenpkg:
	@cd $(MAKEFILE_DIR)/internal/gen && $(MAKE) --quiet test

# Test that `go generate` does not change tables.go
.PHONY: testgenerate
testgenerate: bin/gen
	@if ! $(GEN_TARGET) -dry-run -skip-tests >/dev/null; then \
		$(GEN_TARGET) -dry-run -skip-tests;                   \
	fi;

# Make sure the benchmarks pass (we run them with a short benchtime)
.PHONY: testbenchmarks
testbenchmarks:
	@cd $(MAKEFILE_DIR) && ./scripts/test-benchmarks.bash

# Run all tests (slow)
.PHONY: testall
testall: exhaustive testskipped testgenerate testgenpkg

# Actual ci target (separate because so that we can override GO)
.PHONY: .ci
.ci: GO = $(RICHGO_TARGET)
.ci: export RICHGO_FORCE_COLOR=1
.ci: testverbose
.ci: testbenchmarks

# Run and colorize verbose tests for CI
.PHONY: ci
ci: bin/richgo
ci: vet
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

.PHONY: vet-genpkg
vet-genpkg:
	@cd $(MAKEFILE_DIR)/internal/gen && $(MAKE) --quiet vet

# NOTE: we don't run vet-genpkg here since it requires Go version 1.20
# and we run this against Go 1.19 in CI.
.PHONY: vet
vet: vet-strcase vet-gen

golangci-lint-gen: override GOLANGCI_EXTRA_FLAGS += --build-tags=gen gen.go
golangci-lint-gen: override GOLANGCI_SKIP =

# Run golangci-lint
.PHONY: golangci-lint-strcase golangci-lint-gen
golangci-lint-strcase golangci-lint-gen: bin/golangci-lint
	@$(GOLANGCI_TARGET) run $(GOLANGCI_FLAGS)

.PHONY: golangci-lint
golangci-lint: golangci-lint-strcase golangci-lint-gen

.PHONY: lint
lint: vet golangci-lint golangci-lint-gen

# Make sure there aren't any comments that need addressing (TODO or WARN)
#
# NOTE: not currently part of the "lint" target.
.PHONY: lint-comments
lint-comments:
	@if $(xgrep) $(GREP_COMMENTS) $(COMMENTS); then               \
		echo '';                                                  \
		echo '$(red)FAIL: $(cyan)address comments!$(term-reset)'; \
		exit 1;                                                   \
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

# Run pre-release tests
.PHONY: release
release: exhaustive testinvalid testgenerate lint

# Print information about the version of go being used
.PHONY: env
env:
	@$(GO) env


.PHONY: clean
clean:
	@rm -f cpu*.out mem*.out
	@rm -rf DATA bin
	@$(GO) clean -i -cache
