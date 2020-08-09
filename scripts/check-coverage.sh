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
        awk '//{sub(/(\.[0-9]+)?%$/,"",$3);print$3}'
}

rm_coverage_file() {
    rm $COVERAGE_FILE
}

go test -coverprofile=$COVERAGE_FILE -covermode=atomic ./...
trap rm_coverage_file exit

COVERAGE_LEVEL="$(coverage_level)"
if [ "$COVERAGE_LEVEL" -lt "$MIN_COVERAGE" ]; then
    echo "Coverage ${COVERAGE_LEVEL}% < ${MIN_COVERAGE}% required"
    exit 1
fi
