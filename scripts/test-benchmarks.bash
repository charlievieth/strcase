#!/usr/bin/env bash

set -euo pipefail

if [ -t 1 ]; then
    RED=$'\E[00;31m'
    GREEN=$'\E[00;32m'
    RESET=$'\E[0m'
else
    RED=''
    GREEN=''
    RESET=''
fi

TMP="$(mktemp -t 'strcase.XXXXXX')"
trap 'rm ${TMP}' EXIT

# TODO: test all packages
if ! go test -run '^$' -bench . -benchtime 1ms &>"${TMP}"; then
    echo ''
    cat "${TMP}"
    echo ''
    grep --color=auto --after-context=1 --extended-regexp -- \
        '-+ FAIL:.*' "${TMP}"
    echo ''
    echo "${RED}FAIL:${RESET} error running benchmarks: see above output"
    exit 1
fi
echo "${GREEN}PASS${RESET}"
