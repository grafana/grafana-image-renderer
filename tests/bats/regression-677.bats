#!/usr/bin/env bats

# Regression test for: https://github.com/grafana/grafana-image-renderer/issues/677

load docker

function teardown() {
    _remove_docker
}

@test "entrypoint is wrapped in tini" {
    run _docker inspect --format '{{json .Config.Entrypoint}}' "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^\[\"tini\" ]]
}
