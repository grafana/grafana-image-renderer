#!/usr/bin/env bats

load docker

function teardown() {
    _remove_docker
}

@test "service can create PDF of own health page" {
    CONTAINER_NAME="$(_container_name)"
    run _docker run -p 8081 --health-start-period=1s --health-start-interval=0.1s --name "$CONTAINER_NAME" -d "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [ -n "$output" ]

    # Wait for the container to be healthy
    _wait_for_healthy "$output"
    [ "$status" -eq 0 ]

    # Get port of container
    run _docker container inspect -f '{{json .NetworkSettings.Ports}}' "$CONTAINER_NAME"
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    PORT="$(echo "$output" | jq -r '.["8081/tcp"][0].HostPort')"

    # Execute the request
    run curl -o "$BATS_TEST_TMPDIR/output.pdf" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url=http://localhost:8081/&encoding=pdf"
    [ "$status" -eq 0 ]
}

@test "service can create PNG of own health page" {
    CONTAINER_NAME="$(_container_name)"
    run _docker run -p 8081 --health-start-period=1s --health-start-interval=0.1s --name "$CONTAINER_NAME" -d "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [ -n "$output" ]

    # Wait for the container to be healthy
    _wait_for_healthy "$output"
    [ "$status" -eq 0 ]

    # Get port of container
    run _docker container inspect -f '{{json .NetworkSettings.Ports}}' "$CONTAINER_NAME"
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    PORT="$(echo "$output" | jq -r '.["8081/tcp"][0].HostPort')"

    # Execute the request
    run curl -o "$BATS_TEST_TMPDIR/output.pdf" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url=http://localhost:8081/&encoding=pdf"
    [ "$status" -eq 0 ]
}
