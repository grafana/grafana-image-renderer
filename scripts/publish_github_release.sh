#!/bin/bash

set -e

RELEASE_NOTES=`awk 'BEGIN {FS="##"; RS=""} FNR==2 {print; exit}' CHANGELOG.md`
VERSION=`cat plugin.json|jq '.info.version'| sed s/\"//g`
PRERELEASE=''
LATEST_TAG=$(git describe --tags --abbrev=0)

if [ v${VERSION} == ${LATEST_TAG} ]; then
  echo "Tag ${LATEST_TAG} have already been pushed. Exiting..."
  exit 1
fi

if [[ $VERSION == *"beta"* ]]; then
  PRERELEASE='-prerelease'
fi

git config user.email "eng@grafana.com"
git config user.name "CircleCI Automation"
git tag v${VERSION}
git push origin v${VERSION}
ghr \
  -t ${GITHUB_TOKEN} \
  -u ${CIRCLE_PROJECT_USERNAME} \
  -r ${CIRCLE_PROJECT_REPONAME} \
  -c ${CIRCLE_SHA1} \
  -n "v${VERSION}" \
  -b "${RELEASE_NOTES}" \
  ${PRERELEASE} v${VERSION} ./artifacts/
