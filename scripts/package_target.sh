#!/bin/bash

ARCH="${1:-}"

if [ -z "$ARCH" ]; then
    echo "ARCH (arg 1) has to be set"
    exit 1
fi

node scripts/pkg.js ${ARCH}
node scripts/download_chromium.js ${ARCH}
node scripts/download_grpc.js ${ARCH}
node scripts/rename_executable.js ${ARCH}
cp plugin.json dist/plugin-${ARCH}/
cp README.md dist/plugin-${ARCH}/
