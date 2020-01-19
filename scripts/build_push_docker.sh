#!/bin/bash

set -e

TAG=$(git describe --tags --abbrev=0 | cut -d "v" -f 2)
echo "building ${TAG}"
docker build -t ${IMAGE_NAME}:${TAG} .

if [[ $TAG != *"beta"* ]]; then
  docker tag ${IMAGE_NAME}:${TAG} ${IMAGE_NAME}:latest
fi

echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin
docker push ${IMAGE_NAME}:${TAG}

if [[ $TAG != *"beta"* ]]; then
  docker push ${IMAGE_NAME}:latest
fi
