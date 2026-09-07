#!/bin/sh

set -e

MIN_COVERAGE="$1"

if [ -z "$MIN_COVERAGE" ]; then
    echo "Usage: $0 <min_coverage%> (e.g., $0 80)"
    exit 1
fi

COVERAGE_FILE=coverage.txt

coverage_level() {
    go tool cover -func=$COVERAGE_FILE | \
        grep '^total:' | \
        tee | \
        awk '{ gsub(/%/, "", $3); print $3 }'
}

rm_coverage_file() {
    rm $COVERAGE_FILE
}

go test -coverprofile=$COVERAGE_FILE -covermode=atomic ./...
trap rm_coverage_file exit

COVERAGE_LEVEL="$(coverage_level)"
# Integer compare used to strip the decimal, so 39.9% became 39 and failed
# a 40% gate. Compare as numbers and keep the reported precision.
awk -v c="$COVERAGE_LEVEL" -v m="$MIN_COVERAGE" 'BEGIN {
	if ((c + 0) < (m + 0)) {
		printf "Coverage %s%% < %s%% required\n", c, m
		exit 1
	}
}'
