#!/usr/bin/env bash
set -euo pipefail

HERE="$(dirname "${BASH_SOURCE[0]}")"
if [ -z "$HERE" ]; then
    HERE="."
fi

usage() {
    {
        echo "usage: $0 <docker_image>"
        echo
        echo "Runs all BATS tests for the docker image."
        echo "Requires bats-core to be installed (bats on Debian/Ubuntu/Arch, bats-core on Homebrew)."
    } >&2
    exit 1
}

if [ $# -ne 1 ]; then
    usage
fi

export DOCKER_IMAGE="$1"
if [ -z "${DOCKER_IMAGE:-}" ]; then
    usage
fi

if ! command -v bats &>/dev/null; then
    echo "fatal: bats is not installed" >&2
    exit 1
fi
if ! command -v docker &>/dev/null; then
    echo "fatal: docker is not installed" >&2
    exit 1
fi

PARALLEL=()
if command -v parallel &>/dev/null || command -v rush &>/dev/null; then
    PARALLEL=(--jobs "$(nproc)")
fi

# We want to run the docker-works.bats test first to ensure that the docker image is accessible, and that Docker works.
bats --formatter pretty "$HERE"/docker-works.bats

# Find and run .bats files in $HERE
bats --formatter pretty "${PARALLEL[@]}" --show-output-of-passing-tests --print-output-on-failure "$HERE"
