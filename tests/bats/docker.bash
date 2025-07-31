#!/usr/bin/env bash
# this is a helper script for BATS tests. load it with `load docker`

if [ -z "${DOCKER_IMAGE:-}" ]; then
    echo "fatal: docker image to use in tests must exist in env var \`DOCKER_IMAGE'" >&2
    exit 1
fi

_docker() {
    DOCKER="${DOCKER:-docker}"
    SUDO="${SUDO:-}"
    if [ -z "$SUDO" ]; then
        if ! $DOCKER ps &>/dev/null; then
            SUDO="sudo"
        fi
    fi
    if ! $SUDO "$DOCKER" ps &>/dev/null; then
        echo "fatal: docker isn't running" >&2
        return 1
    fi

    $SUDO "$DOCKER" "$@"
}

CONTAINER_NAME_PREFIX="${CONTAINER_NAME_PREFIX:-image-renderer-bats-}"

_container_name() {
    local random_suffix
    random_suffix=$(tr -dc 'a-z0-9' < /dev/urandom | head -c 8)
    echo "${CONTAINER_NAME_PREFIX}${random_suffix}"
}

_remove_docker() {
    _docker ps -a --filter "name=${CONTAINER_NAME_PREFIX}" --format "{{.ID}}" | while read -r container_id; do
        if [ -n "$container_id" ]; then
            _docker rm -f "$container_id" || true # we don't really mind if we fail to clean up
        fi
    done
}
