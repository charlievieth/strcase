MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKEFILE_DIR  := $(dir $(MAKEFILE_PATH))
PATH          := $(MAKEFILE_DIR)/bin:$(PATH)

COUNT          ?= 1   # Number of benchmark iterations (`-count` flag)
TIMEOUT        ?= 4h  # Benchmark timeout (`-timeout` flag)
BENCHTIME      ?= 1s  # Benchmark run time (`-benchtime` flag)

GO             ?= go
GOBIN          ?= $(MAKEFILE_DIR)/bin
GO_TEST_FLAGS  ?= -timeout=$(TIMEOUT) -count=$(COUNT) -benchtime=$(BENCHTIME)
GO_TEST        ?= $(GO) test $(GO_TEST_FLAGS)
TIMESTAMP      := $(shell date +%s)
TEE            ?= false # Tee benchmark results to a file

BENCHSTAT_PKG  ?= golang.org/x/perf/cmd/benchstat@latest
BENCHCMP_PKG   ?= golang.org/x/tools/cmd/benchcmp@latest

.PHONY: all
all: bench

# install benchstat tool
bin/benchstat:
	@if ! command -v benchstat >/dev/null; then        \
		mkdir -p $(GOBIN);                             \
		GOBIN=$(GOBIN) $(GO) install $(BENCHSTAT_PKG); \
	fi

# install benchcmp tool
bin/benchcmp:
	@if ! command -v benchcmp >/dev/null; then        \
		mkdir -p $(GOBIN);                            \
		GOBIN=$(GOBIN) $(GO) install $(BENCHCMP_PKG); \
	fi

# Install benchstat and benchcmp
.PHONY: install-tools
install-tools: bin/benchstat bin/benchcmp

stdlib: GO_TEST_FLAGS += -stdlib
stdlib-case: GO_TEST_FLAGS += -stdlib-case

# Run benchmarks:
#
#   bench:       run strcase target
#   strcase:     strcase package
#   stdlib:      strings package
#   stdlib-case: strings package and covert case
#
.PHONY: bench strcase stdlib stdlib-case
bench strcase stdlib stdlib-case:
	@if $(TEE); then                                       \
		$(GO_TEST) -bench . | tee "$(@).$(TIMESTAMP).txt"; \
	else                                                   \
		$(GO_TEST) -bench .;                               \
	fi;

.PHONY: benchstat
benchstat:
	@benchstat -confidence=0 stdlib.$(TIMESTAMP).txt strcase.$(TIMESTAMP).txt | \
		tee result.$(TIMESTAMP).txt

# release creates the "release" benchmarks which compare the
# performance of strcase against the stdlib's strings package
#
# WARN: this is very very slow
.PHONY: release
release: COUNT = 6
release: TEE = true
release: bin/benchcmp commit_hash git_diff
release: strcase # The following targets need to be ran sequentially
release: stdlib
release: benchstat

# Since the release target compares the results of the strcase and stdlib
# targets it cannot be ran in parallel.
.NOTPARALLEL: release

.PHONY: commit_hash
commit_hash:
	@git rev-parse HEAD > commit.$(TIMESTAMP).txt

.PHONY: git_diff
git_diff:
	@git diff --patch > patch.$(TIMESTAMP).patch

# test is a a no-op and is only here to make `checkmake` happy
.PHONY: test
test:
	@$(GO) test

.PHONY: clean
clean:
	@[[ ! -d $(GOBIN) ]] || rm -r $(GOBIN);
	@$(GO) clean;
