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

echo '# Testing benchmarks'

# Go packages
readarray -t PACKAGES < <(go list ./...)

TMP="$(mktemp -d -t 'strcase.XXXXXX')"
trap 'rm -r "${TMP}"' EXIT

EXIT_CODE=0

for pkg in "${PACKAGES[@]}"; do
    # echo "# ${pkg}"
    out="${TMP}/${pkg//\//_}"
    if ! go test -run '^$' -shuffle on -bench . -benchtime 10us "${pkg}" &>"${out}"; then
        echo ''
        cat "${out}"
        echo ''
        echo "# ${pkg}"
        grep --color=auto --after-context=1 --extended-regexp -- \
            '-+ FAIL:.*' "${out}"
        echo ''
        printf '%sFAIL%s\t%s\n' "${RED}" "${RESET}" "${pkg}"
        EXIT_CODE=1
    else
        printf '%sok%s\t%s\n' "${GREEN}" "${RESET}" "${pkg}"
    fi
done

exit $EXIT_CODE
