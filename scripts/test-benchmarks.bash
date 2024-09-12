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
if ((${#PACKAGES[@]} == 0)); then
    printf '%sFAIL%s\t%s\n' "${RED}" "${RESET}" 'no Go packages'
    exit 1
fi

TMP="$(mktemp -d -t 'strcase.XXXXXX')"
trap 'rm -r "${TMP}"' EXIT

for pkg in "${PACKAGES[@]}"; do
    out="${TMP}/${pkg//\//_}"
    # Run tests in parallel in a sub-shell
    (
        if ! go test -run '^$' -shuffle on -bench . -benchtime 1us "${pkg}" &>"${out}"; then
            echo ''
            cat "${out}"
            echo ''
            echo "# ${pkg}"
            \grep --color=auto --after-context=1 --extended-regexp -- \
                '-+ FAIL:.*' "${out}"
            echo ''
            printf '%sFAIL%s\t%s\n' "${RED}" "${RESET}" "${pkg}"
            touch "${TMP}/fail"
        else
            printf '%sok%s\t%s\n' "${GREEN}" "${RESET}" "${pkg}"
        fi
    ) &
done

wait

if [[ -f "${TMP}/fail" ]]; then
    exit 1
fi
exit 0
