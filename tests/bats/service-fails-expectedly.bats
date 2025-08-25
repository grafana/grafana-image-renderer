#!/usr/bin/env bats

load docker

function teardown() {
    _remove_docker
}

@test "service fails requests with no auth token" {
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
    run curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/render?url=http://localhost:8081/&encoding=pdf"
    [ "$status" -eq 0 ]
    [ "$output" = "401" ]
}

@test "service fails requests with empty auth token" {
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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token:' "http://localhost:$PORT/render?url=http://localhost:8081/&encoding=pdf"
    [ "$status" -eq 0 ]
    [ "$output" = "401" ]
}

@test "service fails requests with whitespace auth token" {
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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token:     ' "http://localhost:$PORT/render?url=http://localhost:8081/&encoding=pdf"
    [ "$status" -eq 0 ]
    [ "$output" = "401" ]
}

@test "service fails requests with wrong auth token" {
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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: wrong' "http://localhost:$PORT/render?url=http://localhost:8081/&encoding=pdf"
    [ "$status" -eq 0 ]
    [ "$output" = "401" ]
}

@test "service fails requests with untrusted URL" {
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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url=file:///etc/passwd&encoding=pdf"
    [ "$status" -eq 0 ]
    [ "$output" = "403" ]
}

@test "service fails requests with socket URL" {
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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url=socket:///tmp/socket&encoding=pdf"
    [ "$status" -eq 0 ]
    [ "$output" = "403" ]
}

@test "service fails requests with missing URL" {
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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: -' "http://localhost:$PORT/render"
    [ "$status" -eq 0 ]
    [ "$output" = "400" ]
}

@test "service fails requests with empty URL" {
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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url="
    [ "$status" -eq 0 ]
    [ "$output" = "400" ]
}

@test "service fails requests with invalid width" {
    skip "node.js does not fail this?"

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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url=http://localhost:8081/&width=text"
    [ "$status" -eq 0 ]
    [ "$output" = "400" ]
}

@test "service fails requests with invalid height" {
    skip "node.js does not fail this?"

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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url=http://localhost:8081/&height=text"
    [ "$status" -eq 0 ]
    [ "$output" = "400" ]
}

@test "service fails requests with invalid encoding" {
    skip "node.js does not fail this?"

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
    run curl -s -o /dev/null -w "%{http_code}" -H 'X-Auth-Token: -' "http://localhost:$PORT/render?url=http://localhost:8081/&encoding=xyz"
    [ "$status" -eq 0 ]
    [ "$output" = "400" ]
}
