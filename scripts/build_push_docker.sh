#!/bin/bash

set -e

PLATFORM=linux/amd64,linux/arm64

TAG=''
if [ "$1" = "master" ]; then
  TAG="master-$(git rev-parse --short HEAD)"
else
  git fetch --tags
  TAG=$(git describe --tags --abbrev=0 | cut -d "v" -f 2)
fi

# echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin

BUILDER="$(docker buildx create --use)"
# BUILDX_BUILD_OPTS=(--platform "$PLATFORM" --builder "$BUILDER" --push -t "${IMAGE_NAME}:${TAG}")
BUILDX_BUILD_OPTS=(--platform "$PLATFORM" --builder "$BUILDER" -t "${IMAGE_NAME}:${TAG}")

if [ -z "$(echo $TAG | grep -E "beta|master")" ]; then
  BUILDX_BUILD_OPTS+=(-t "${IMAGE_NAME}:latest")
fi

docker run --rm --privileged docker.io/multiarch/qemu-user-static --reset -p yes
docker buildx build "${BUILDX_BUILD_OPTS[@]}" .
docker buildx rm "$BUILDER"
