#!/bin/bash

ARCH="${1:-}"
SKIP_CHROMIUM=${2:-false}
OUT="${3:-}"

if [ -z "$ARCH" ]; then
    echo "ARCH (arg 1) has to be set"
    exit 1
fi

mkdir -p dist
node scripts/pkg.js ${ARCH} ${OUT}
if [ $? != 0 ]; then
   echo "${?}\n". 1>&2 && exit 1
fi

if [ ${SKIP_CHROMIUM} = false ]; then
   node scripts/download_chrome.js ${ARCH} ${OUT}
else
    echo "Skipping chrome download"
fi

if [ $? != 0 ]; then
   echo "${?}\n". 1>&2 && exit 1
fi

node scripts/rename_executable.js ${ARCH} ${OUT}

if [ $? != 0 ]; then
   echo "${?}\n". 1>&2 && exit 1
fi

COPY_PATH=dist/plugin-${ARCH}/

if [ ! -z "$OUT"  ]; then
    COPY_PATH=dist/${OUT}
fi

cp plugin.json ${COPY_PATH}
cp README.md ${COPY_PATH}
cp CHANGELOG.md ${COPY_PATH}
cp LICENSE ${COPY_PATH}
cp -r img ${COPY_PATH}
