#!/bin/bash

set -e

git fetch --tags

RELEASE_NOTES=$(awk 'BEGIN {FS="##"; RS=""} FNR==1 {print; exit}' CHANGELOG.md)
VERSION=$(cat plugin.json|jq '.info.version'| sed s/\"//g)
PRERELEASE=''
LATEST_TAG=$(git describe --tags --abbrev=0)

if [ v"${VERSION}" == "${LATEST_TAG}" ]; then
  echo "Tag ${LATEST_TAG} have already been pushed. Exiting..."
  exit 1
fi

if [[ $VERSION == *"beta"* ]]; then
  PRERELEASE='-prerelease'
fi

git config user.email "eng@grafana.com"
git config user.name "Drone Automation"

echo "Pushing tag v${VERSION}..."
git tag v"${VERSION}"
git push "https://${GITHUB_APP_ID}:${GITHUB_TOKEN}@github.com/grafana/grafana-image-renderer.git" v"${VERSION}"

echo "Pushing multiple artifacts to release v${VERSION}..."
ghr \
  -u "${DRONE_REPO_OWNER}" \
  -r "${DRONE_REPO_NAME}" \
  -c "${DRONE_COMMIT_SHA}" \
  -n "v${VERSION}" \
  -b "${RELEASE_NOTES}" \
  ${PRERELEASE} v"${VERSION}" ./artifacts/
