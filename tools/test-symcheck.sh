#!/bin/bash
# Tests tools/symcheck.py against the fixtures in tools/testdata/symcheck/.
set -uo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
DATA=$HERE/testdata/symcheck
failures=0

expect_exit() {
    local description=$1 expected=$2
    shift 2
    python3 -I "$HERE/symcheck.py" "$@" >/dev/null 2>&1
    local actual=$?
    if [ "$actual" = "$expected" ]; then echo "ok   - $description"; else echo "FAIL - $description (exit $actual, wanted $expected)"; failures=$((failures + 1)); fi
}

expect_exit "optional symbol missing is not fatal" 0 --symbols "$DATA/symbols.yaml" --fw-dir "$DATA/fw"
expect_exit "required symbol missing in a build fails" 1 --symbols "$DATA/symbols-required-missing.yaml" --fw-dir "$DATA/fw"
expect_exit "missing dump is an input error" 2 --symbols "$DATA/symbols.yaml" --fw-dir "$DATA/fw" --builds 9.9.9
expect_exit "missing symbols file is an input error" 2 --symbols "$DATA/nope.yaml" --fw-dir "$DATA/fw"
python3 -I "$HERE/symcheck.py" --symbols "$DATA/symbols.yaml" --fw-dir "$DATA/fw" 2>&1 | grep -q 'MISSING' &&
    echo "ok   - matrix marks the missing optional symbol" || { echo "FAIL - matrix shows no MISSING"; failures=$((failures + 1)); }

[ "$failures" -eq 0 ] && echo "all checks passed" || { echo "$failures check(s) failed"; exit 1; }
