#!/usr/bin/env bats

# Regression test for: https://github.com/grafana/grafana-image-renderer/issues/694

load docker

function teardown() {
    _remove_docker
}

@test "openshift: docker image starts with random UID" {
    # We want the container to start and be healthy.
    RANDOM_UID="$(shuf -i 100000-999999 -n 1)"
    # https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html/images/creating-images#use-uid_create-images
    #   > Because the container user is always a member of the root group, the container user can read and write these files.
    # https://www.redhat.com/en/blog/a-guide-to-openshift-and-uids
    #   > Notice the Container is using the UID from the Namespace. An important detail to notice is that the user in the Container always has GID=0, which is the root group. 
    run _docker run --user "$RANDOM_UID":0 --health-start-period=1s --health-start-interval=0.1s --name "$(_container_name)" -d "$DOCKER_IMAGE"
    [ "$status" -eq 0 ]
    [ -n "$output" ]

    # Wait for the container to be healthy
    _wait_for_healthy "$output"
    [ "$status" -eq 0 ]
}
