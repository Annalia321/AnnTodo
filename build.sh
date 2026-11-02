#!/usr/bin/env sh
# Build AnnTodo release binaries.
#   ./build.sh                 # current platform -> ./anntodo
#   ./build.sh 1.0.0 all       # windows/linux amd64+arm64 -> ./dist/
set -eu
cd "$(dirname "$0")"
VERSION="${1:-dev}"
MODE="${2:-}"

(cd web && { [ -d node_modules ] || npm ci; } && npm run build)

export CGO_ENABLED=0
LDFLAGS="-s -w -X anntodo/pkg/version.Version=${VERSION}"

if [ "$MODE" != "all" ]; then
  go build -trimpath -ldflags "$LDFLAGS" -o anntodo ./cmd/anntodo
  echo "built ./anntodo (${VERSION})"
  exit 0
fi

for t in windows/amd64 windows/arm64 linux/amd64 linux/arm64; do
  os="${t%/*}"; arch="${t#*/}"
  out="dist/anntodo-${VERSION}-${os}-${arch}"
  mkdir -p "$out/conf"
  bin=anntodo; [ "$os" = windows ] && bin=anntodo.exe
  GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$LDFLAGS" -o "$out/$bin" ./cmd/anntodo
  cp conf/anntodo.ini "$out/conf/"
  cp LICENSE README.md "$out/"
  [ "$os" = linux ] && cp docker/anntodo.service "$out/"
  echo "built $out"
done
