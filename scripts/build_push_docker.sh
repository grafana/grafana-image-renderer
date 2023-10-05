#!/bin/bash

set -e

TAG=''
if [ "$1" = "master" ]; then
  TAG="master-$(git rev-parse --short HEAD)"
else
  git fetch --tags
  TAG=$(git describe --tags --abbrev=0 | cut -d "v" -f 2)
fi

echo "building ${TAG}"
echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin
tags=()
tags+=${IMAGE_NAME}:${TAG}
if [ -z "$(echo $TAG | grep -E "beta|master")" ]; then
  tags+={IMAGE_NAME}:latest
fi

docker buildx build --platform linux/amd64,linux/arm64 --push -t ${tags[@]} .
