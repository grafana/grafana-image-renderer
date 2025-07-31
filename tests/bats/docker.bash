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
            if command -v sudo &>/dev/null; then
                SUDO="sudo"
            elif command -v doas &>/dev/null; then
                SUDO="doas"
            else
                echo "error: docker seemingly requires elevated privileges but sudo and doas are not available" >&2
                return 1
            fi
        fi
    fi
    if ! $SUDO "$DOCKER" ps &>/dev/null; then
        echo "error: docker isn't running" >&2
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
            echo "info: removed container $container_id"
        fi
    done
}

_wait_for_healthy() {
    if [ $# -lt 1 ]; then
        echo "error: _wait_for_healthy requires a container name" >&2
        return 1
    fi
    CONTAINER_NAME="$1"
    TIMEOUT="${2:-10}" # default timeout is 10 seconds
    START_TIME="$(date +%s)"
    END_TIME=$((START_TIME + TIMEOUT))
    while [ "$(date +%s)" -lt "$END_TIME" ]; do
        if _docker inspect --format '{{.State.Health.Status}}' "$CONTAINER_NAME" | grep -q 'healthy'; then
            return 0
        fi
        sleep 0.2
    done
    echo "error: container $CONTAINER_NAME did not become healthy within $TIMEOUT seconds" >&2
    return 1
}
