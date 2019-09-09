#!/bin/bash

ARCH="${1:-}"

if [ -z "$ARCH" ]; then
    echo "ARCH (arg 1) has to be set"
    exit 1
fi

mkdir -p artifacts
zip -yqr artifacts/plugin-${ARCH}.zip dist/plugin-${ARCH}
