#!/usr/bin/env bash
set -euo pipefail

ARCH="${1:-}"
SKIP_CHROMIUM=${2:-false}
OUT="${3:-}"
usage() {
   echo "usage: $0 <arch> [skip_chromium: true|false] [out: path]" >&2
}

if [ -z "$ARCH" ]; then
    usage
    exit 1
fi
if [ "$SKIP_CHROMIUM" != "true" ] && [ "$SKIP_CHROMIUM" != "false" ]; then
    usage
    exit 1
fi

mkdir -p dist
node scripts/pkg.js "${ARCH}" "${OUT}"

if [ "${SKIP_CHROMIUM}" = "false" ]; then
   node scripts/download_chrome.js "${ARCH}" "${OUT}"
else
   echo "Skipping chrome download"
fi

node scripts/rename_executable.js "${ARCH}" "${OUT}"

COPY_PATH=dist/plugin-"${ARCH}"/
if [ -n "$OUT"  ]; then
   COPY_PATH=dist/"${OUT}"
fi

cp plugin.json "${COPY_PATH}"
cp README.md "${COPY_PATH}"
cp CHANGELOG.md "${COPY_PATH}"
cp LICENSE "${COPY_PATH}"
cp -r img "${COPY_PATH}"
