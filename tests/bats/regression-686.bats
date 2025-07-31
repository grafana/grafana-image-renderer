#!/usr/bin/env bats

# Regression test for: https://github.com/grafana/grafana-image-renderer/issues/686

load docker

function teardown() {
    _remove_docker
}

@test "kubernetes behavior: user is set to numeric UID" {
    # When using Kubernetes with `.spec.containers[].securityContext.runAsNonRoot: true`, the Dockerfile must set a numeric user ID.
    run _docker image inspect --format '{{json .Config.User}}' "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^\"[0-9]+\"$ ]]
}

@test "UID is same as nonroot user" {
    # We need the UID we select to equal that of the actual user "nonroot"
    run _docker image inspect --format '{{json .Config.User}}' "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    IMG_USER="${output//\"/}" # strip the quotes around the JSON value
    run _docker run --entrypoint sh --user nonroot --name "$(_container_name)" "$DOCKER_IMAGE" -c 'id -u'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "$IMG_USER" ]
}
