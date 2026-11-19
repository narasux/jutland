#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export GOCACHE="${GOCACHE:-/tmp/jutland-gocache}"

echo "== go test ./... =="
go test ./...
echo "== go vet ./... =="
go vet ./...
echo "== go build ./... =="
go build ./...
