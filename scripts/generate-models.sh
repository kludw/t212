#!/bin/bash
set -euo pipefail

SCHEMA_URL="https://docs.trading212.com/_bundle/api.yaml?download"
SCHEMA_FILE="api.yaml"
OUTPUT_FILE="models.gen.go"
OAPI_CODEGEN_PKG="github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"

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

ensure_oapi_codegen() {
    if command -v oapi-codegen >/dev/null 2>&1; then
        return
    fi

    echo "oapi-codegen not found, installing latest..."
    go install "${OAPI_CODEGEN_PKG}@latest"

    local gobin
    gobin="$(go env GOBIN)"
    if [ -z "$gobin" ]; then
        gobin="$(go env GOPATH)/bin"
    fi
    export PATH="$gobin:$PATH"
}

download_schema() {
    echo "Downloading Trading212 schema..."
    curl -fsSL -o "$SCHEMA_FILE" "$SCHEMA_URL"
}

generate_models() {
    echo "Generating models from Trading212 schema..."
    oapi-codegen \
        -generate types \
        -package t212 \
        -o "$OUTPUT_FILE" \
        "$SCHEMA_FILE"
}

ensure_go
ensure_repo_root
ensure_oapi_codegen
download_schema
generate_models

echo "Done. Wrote $OUTPUT_FILE"
