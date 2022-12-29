MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILE_DIR  := $(dir $(MAKEFILE_PATH))

# Test options
GO_TEST_ARGS ?= -cover -covermode=count
GO_TEST ?= go test $(GO_TEST_ARGS)
COMMENTS ?= 'TODO|WARN|FIXME|CEV'
GREP ?= \grep --color=always

# Benchmark options
NO_TESTS = ^$
INDEX_BENCHMARKS = ^BenchmarkIndex('\$'|Hard|Torture|Periodic(Unicode)?)

# Color support.
red = $(shell { tput setaf 1 || tput AF 1; } 2>/dev/null)
yellow = $(shell { tput setaf 3 || tput AF 3; } 2>/dev/null)
cyan = $(shell { tput setaf 6 || tput AF 6; } 2>/dev/null)
term-reset = $(shell { tput sgr0 || tput me; } 2>/dev/null)

.PHONY: all
all: test

.PHONY: test
test:
	@$(GO_TEST)

# Run short tests
.PHONY: short
short: GO_TEST_ARGS += -short
short:
	$(GO_TEST)

# Run exhaustive fuzz tests
.PHONY: exhaustive
exhaustive:
	@$(GO_TEST) -exhaustive

# Assert that there are no skipped tests
.PHONY: skipped_tests
skipped_tests:
	@if $(GO_TEST) -v | $(GREP) --fixed-strings -- '--- SKIP:'; then \
		echo '$(red)FAIL: $(cyan)skipped tests$(term-reset)';        \
		exit 1;                                                      \
	fi

# Calibrate brute-force cutover
.PHONY: calibrate
calibrate:
	@$(GO_TEST) -run TestCalibrate -calibrate

# Run all tests (slow)
.PHONY: test_all
test_all: test exhaustive skipped_tests

.PHONY: vet
vet:
	@go vet

.PHONY: golangci-lint
golangci-lint:
	@# TODO: do we want to use MAKEFILE_DIR
	@if command -v golangci-lint >/dev/null; then                       \
		golangci-lint run $(MAKEFILE_DIR);                              \
	else                                                                \
		echo '$(yellow)WARN:$(term-reset) golangci-lint not installed'; \
	fi

.PHONY: lint_comments
lint_comments:
	@if $(GREP) --line-number --extended-regexp $(COMMENTS) *.go; then \
		echo '';                                                       \
		echo '$(yellow)WARN: $(cyan)address comments!$(term-reset)';   \
		exit 1;                                                        \
	fi

.PHONY: lint
lint: vet golangci-lint

# TODO: remove?
.PHONY: pretty_test
pretty_test: GO_TEST = richgo test

# TODO: remove?
.PHONY: watch
watch:
	gotestsum --watch -- github.com/charlievieth/strcase

.PHONY: clean
clean:
	rm -f ./gen ./strcase.test ./cpu.out

# grep -E 'TODO|WARN|CEV' *.go

# .PHONY: bench_index
# bench_index:
# 	go test -run "$(NO_TESTS)" -bench "$(INDEX_BENCHMARKS)"
