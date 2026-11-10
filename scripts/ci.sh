#!/usr/bin/env bash
# Local and CI checks: tidy, formatting, vet, staticcheck, tests, build.
# Set SKIP_STATICCHECK=1 on Go toolchains older than 1.26 (staticcheck needs it).
set -euo pipefail

STATICCHECK_VERSION="v0.8.2"

go mod tidy

unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "gofmt: folgende Dateien sind nicht formatiert (gofmt -w ausführen):"
  echo "$unformatted"
  exit 1
fi

go vet ./...

if [ "${SKIP_STATICCHECK:-}" != "1" ]; then
  go run "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}" ./...
fi

go test ./...
go build -o /dev/null .

echo "CI checks passed."
