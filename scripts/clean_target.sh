#!/bin/bash

ARCH="${1:-}"
OUT="${2:-}"

if [ -z "$ARCH" ]; then
    echo "ARCH (arg 1) has to be set"
    exit 1
fi

PLUGIN_NAME=plugin-${ARCH}

if [ ! -z "$OUT"  ]; then
    PLUGIN_NAME=${OUT}
fi

rm -rf .dist/${PLUGIN_NAME}
rm -f ./artifacts/${PLUGIN_NAME}.zip