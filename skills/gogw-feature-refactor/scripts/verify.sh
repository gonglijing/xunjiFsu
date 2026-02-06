#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT_DIR"

echo "[1/3] go build (core packages + cmd)"
GOCACHE=/tmp/gocache go build ./internal/handlers ./internal/driver ./internal/app ./cmd/...

echo "[2/3] frontend build"
npm --prefix ui/frontend run build

echo "[3/3] mini binary build"
GOCACHE=/tmp/gocache make build-mini

echo "done"
