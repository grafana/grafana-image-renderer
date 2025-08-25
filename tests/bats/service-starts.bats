#!/usr/bin/env bats

load docker

function teardown() {
    _remove_docker
}

@test "docker image has wget" {
    # wget is used in the healthcheck we ship.
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'command -v wget'
    [ "$status" -eq 0 ]
    [ -n "$output" ]
}

@test "docker image starts" {
    # We want the container to start and be healthy.
    run _docker run --health-start-period=1s --health-start-interval=0.1s --name "$(_container_name)" -d "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [ -n "$output" ]

    # Wait for the container to be healthy
    _wait_for_healthy "$output"
    [ "$status" -eq 0 ]
}
