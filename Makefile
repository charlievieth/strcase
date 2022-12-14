NO_TESTS = ^$
INDEX_BENCHMARKS = ^BenchmarkIndex('\$'|Hard|Torture|Periodic(Unicode)?)
INDEX_TESTS = ^TestIndex(Case|Fuzz|Unicode|XXX)?\$

# Color support.
red = $(shell { tput setaf 1 || tput AF 1; } 2>/dev/null)
cyan = $(shell { tput setaf 6 || tput AF 6; } 2>/dev/null)
term-reset = $(shell { tput sgr0 || tput me; } 2>/dev/null)

.PHONY: run_tests
run_tests:
	go test

.PHONY: skipped_tests
skipped_tests:
	@if go test -v | grep --color=always --fixed-strings -- '--- SKIP:'; then \
		echo '$(red)Error: $(cyan)skipped tests$(term-reset)'; \
		exit 1; \
	fi

.PHONY: test
test: run_tests skipped_tests

.PHONY: test_index
test_index:
	@richgo test -run "$(INDEX_TESTS)" github.com/charlievieth/strcase

.PHONY: exhaustive
exhaustive: run_tests
	@go test -exhaustive

# Calibrate brute-force cutover
.PHONY: calibrate
calibrate:
	@go test -run TestCalibrate -calibrate

.PHONY: vet
vet:
	@go vet

.PHONY: golangci-lint
golangci-lint:
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo '$(red)Error:$(term-reset) golangci-lint not installed'; \
	fi

.PHONY: lint
lint: vet golangci-lint

.PHONY: watch
watch:
	gotestsum --watch -- github.com/charlievieth/strcase

# gotestsum -- -run "$(INDEX_TESTS)" github.com/charlievieth/strcase

.PHONY: bench_index
bench_index:
	go test -run "$(NO_TESTS)" -bench "$(INDEX_BENCHMARKS)"
