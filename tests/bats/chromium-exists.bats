#!/usr/bin/env bats

load docker

function teardown() {
    _remove_docker
}

@test "chromium is installed in CHROME_BIN's location" {
    # Extract the CHROME_BIN variable
    # shellcheck disable=SC2016 # we intentionally use single quotes to avoid variable expansion
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'echo $CHROME_BIN'
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    CHROME_BIN="$output"

    # Check if the file exists
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c "test -f '$CHROME_BIN'"
    [ "$status" -eq 0 ]

    # Check if the file is executable
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c "test -x '$CHROME_BIN'"
    [ "$status" -eq 0 ]

    # Check it returns its version
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c "'$CHROME_BIN' --version"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^Chromium\ [0-9]+ ]]
}
