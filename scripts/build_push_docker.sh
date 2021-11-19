#!/bin/bash

set -e

TAG=''
if [[ "$1" == "master" ]]; then
  TAG="master-$(git rev-parse --short HEAD)"
else
  TAG=$(git describe --tags --abbrev=0 | cut -d "v" -f 2)
fi

echo "building ${TAG}"
docker build -t ${IMAGE_NAME}:${TAG} .

echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin
docker push ${IMAGE_NAME}:${TAG}

if [[ $TAG != *"beta"* ]] && [[ $TAG != *"master"* ]]; then
  docker tag ${IMAGE_NAME}:${TAG} ${IMAGE_NAME}:latest
  docker push ${IMAGE_NAME}:latest
fi
