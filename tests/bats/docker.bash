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

CONTAINER_NAME_PREFIX="${CONTAINER_NAME_PREFIX:-image-renderer-bats}"

_container_name() {
    local random_suffix
    random_suffix=$(tr -dc 'a-z0-9' < /dev/urandom | head -c 8)
    echo "${CONTAINER_NAME_PREFIX}-${BATS_SUITE_TEST_NUMBER}-${random_suffix}"
}

_remove_docker() {
    # The trailing dash is important to avoid removing unrelated containers.
    # E.g. prefix-2 would match prefix-20-abc, but prefix-2- would not.
    _docker ps -a --filter "name=${CONTAINER_NAME_PREFIX}-${BATS_SUITE_TEST_NUMBER}-" --format "{{.ID}}" | while read -r container_id; do
        if [ -n "$container_id" ]; then
            _docker kill "$container_id" &>/dev/null || true
            _docker rm -f "$container_id" &>/dev/null || true # we don't really mind if we fail to clean up
            echo "info: removed container $container_id" >&2
        fi
    done
}

_wait_for_healthy() {
    if [ $# -lt 1 ]; then
        echo "error: _wait_for_healthy requires a container name" >&2
        return 1
    fi
    CONTAINER_NAME="$1"
    TIMEOUT="${2:-30}"
    # Stupid simple way to check timeouts. The docker commands are practically instant, so we just *5 (as we sleep 0.2s at a time) and iterate.
    for _i in $(seq 1 $((TIMEOUT * 5))); do
        if [ "$(_docker inspect --format '{{.State.Running}}' "$CONTAINER_NAME")" != "true" ]; then
            echo "error: container $CONTAINER_NAME is not running" >&2
            docker logs "$CONTAINER_NAME" >&2 || true # print the container log if possible.
            return 1
        fi
        if [ "$(_docker inspect --format '{{.State.Health.Status}}' "$CONTAINER_NAME")" = "healthy" ]; then
            return 0
        fi
        sleep 0.2
    done
    echo "error: container $CONTAINER_NAME did not become healthy within $TIMEOUT seconds" >&2
    docker logs "$CONTAINER_NAME" >&2 || true # print the container log if possible.
    return 1
}
