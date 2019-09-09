#!/bin/bash

ARCH="${1:-}"

if [ -z "$ARCH" ]; then
    echo "ARCH (arg 1) has to be set"
    exit 1
fi

mkdir -p dist
node scripts/pkg.js ${ARCH}
if [ $? != 0 ]; then
   echo "${?}\n". 1>&2 && exit 1
fi

node scripts/download_chromium.js ${ARCH}

if [ $? != 0 ]; then
   echo "${?}\n". 1>&2 && exit 1
fi

node scripts/download_grpc.js ${ARCH}

if [ $? != 0 ]; then
   echo "${?}\n". 1>&2 && exit 1
fi

node scripts/rename_executable.js ${ARCH}

if [ $? != 0 ]; then
   echo "${?}\n". 1>&2 && exit 1
fi

cp plugin.json dist/plugin-${ARCH}/
cp README.md dist/plugin-${ARCH}/
cp LICENSE dist/plugin-${ARCH}/
