# vim: ts=4 sw=4

MAKEFILE_PATH := $(realpath $(lastword $(MAKEFILE_LIST)))
MAKEFILE_DIR  := $(abspath $(dir $(MAKEFILE_PATH)))

# Run tests and linters. If this passes then CI tests
# should also pass.
.PHONY: all
all: test lint

# Common variables
include $(MAKEFILE_DIR)/../../common.mk

# # Run tests an
# Run verbose tests
testverbose: override GO_TEST_FLAGS += -v

# Run verbose tests
testrace: override GO_COVER_FLAGS =
testrace: override GO_TEST_FLAGS += -race

.PHONY: test testverbose testrace
test testverbose testrace:
	@GOGC=$(GO_GOGC) $(GO_TEST) ./...

# Actual ci target (separate because so that we can override GO)
.PHONY: .ci
.ci: GO = $(RICHGO_TARGET)
.ci: export RICHGO_FORCE_COLOR=1
.ci: testverbose

# Run and colorize verbose tests for CI
.PHONY: ci
ci: bin/richgo
ci: .ci

# Run golangci-lint
.PHONY: golangci-lint
golangci-lint: bin/golangci-lint
	@$(GOLANGCI_TARGET) run

.PHONY: vet
vet:
	@$(GO) vet ./...

.PHONY: lint
lint: vet golangci-lint

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

.PHONY: clean
clean:
	@$(GO) clean -i ./...
