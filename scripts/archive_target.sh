#!/bin/bash

ARCH="${1:-}"

if [ -z "$ARCH" ]; then
    echo "ARCH (arg 1) has to be set"
    exit 1
fi

OUT="${2:-plugin-${ARCH}}"

mkdir -p artifacts
(cd dist && zip -yqr ../artifacts/${OUT}.zip ${OUT})