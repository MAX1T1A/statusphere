#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

VERSION="${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo dev)}"
OUT="${OUT:-dist}"
LDFLAGS="-s -w -X statusphere-client/internal/version.Version=${VERSION}"

mkdir -p "$OUT"
for arch in amd64 arm64; do
    CGO_ENABLED=0 GOOS=linux GOARCH="$arch" \
        go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/statusphere-linux-$arch" ./cmd/client
    echo "built $OUT/statusphere-linux-$arch  ($VERSION)"
done
