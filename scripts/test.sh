#!/bin/bash
set -euo pipefail

ensure_go() {
    if ! command -v go >/dev/null 2>&1; then
        echo "error: go is not installed. Install Go from https://go.dev/dl/ and retry." >&2
        exit 1
    fi
}

ensure_repo_root() {
    if [ "$(basename "$PWD")" = "scripts" ]; then
        cd ..
    fi
}

run_tests() {
    echo "Running tests..."
    go test -race -count=1 ./...
}

ensure_go
ensure_repo_root
run_tests

echo "Done."
