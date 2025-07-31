#!/usr/bin/env bats

load docker

function teardown() {
    _remove_docker
}

@test "docker is accessible from the CLI" {
    run _docker ps &>/dev/null
}

@test "docker image exists" {
    run _docker images -q "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [ -n "$output" ]
}
