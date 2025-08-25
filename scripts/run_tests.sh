#!/usr/bin/env bash
set -euo pipefail

# Install / update dependencies
yarn install --frozen-lockfile

# Start Grafana
docker compose -f ./devenv/docker/test/docker-compose.yaml up -d --wait
cleanup() {
  docker compose -f ./devenv/docker/test/docker-compose.yaml down --remove-orphans
}
trap cleanup EXIT

# Start testing
yarn jest
