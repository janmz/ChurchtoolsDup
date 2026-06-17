#!/usr/bin/env bash
# Embeds vaya.ico (project root, or assets/vaya.ico) into a Windows .syso resource for go build.
set -euo pipefail

ICON=""
if [ -f vaya.ico ]; then
  ICON="vaya.ico"
elif [ -f assets/vaya.ico ]; then
  ICON="assets/vaya.ico"
else
  echo "No vaya.ico found; skipping Windows icon embed."
  exit 0
fi

VERSION="${1:-0.0.0}"
TARGET_GOOS="${GOOS:-windows}"
TARGET_GOARCH="${GOARCH:-amd64}"

# go-winres must be built for the CI host (linux/darwin). With GOOS=windows,
# go install produces go-winres.exe which is not runnable on Linux runners.
GOOS= GOARCH= CGO_ENABLED= go install github.com/tc-hib/go-winres@v0.3.3

GO_WINRES=""
if bin="$(go env GOBIN 2>/dev/null)" && [ -n "$bin" ] && [ -x "$bin/go-winres" ]; then
  GO_WINRES="$bin/go-winres"
elif [ -x "$(go env GOPATH)/bin/go-winres" ]; then
  GO_WINRES="$(go env GOPATH)/bin/go-winres"
elif command -v go-winres >/dev/null 2>&1; then
  GO_WINRES="$(command -v go-winres)"
fi

if [ -z "$GO_WINRES" ]; then
  echo "go-winres not found after go install (GOPATH=$(go env GOPATH), GOBIN=$(go env GOBIN))" >&2
  exit 1
fi

export GOOS="$TARGET_GOOS"
export GOARCH="$TARGET_GOARCH"

"$GO_WINRES" simply \
  --icon "$ICON" \
  --manifest cli \
  --file-version "$VERSION" \
  --product-version "$VERSION"

echo "Embedded $ICON for ${GOOS}/${GOARCH}"
