#!/usr/bin/env bats

# Regression test for: https://github.com/grafana/grafana-image-renderer/issues/694

load docker

function teardown() {
    _remove_docker
}

@test "openshift: docker image starts with random UID" {
    # We want the container to start and be healthy.
    RANDOM_UID="$(shuf -i 100000-999999 -n 1)"
    run _docker run --user "$RANDOM_UID":"$RANDOM_UID" --health-start-period=1s --health-start-interval=0.1s --name "$(_container_name)" --rm -d "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [ -n "$output" ]

    # Wait for the container to be healthy
    _wait_for_healthy "$output"
    [ "$status" -eq 0 ]
}
