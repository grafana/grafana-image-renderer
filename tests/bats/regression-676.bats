#!/usr/bin/env bats

# Regression test for: https://github.com/grafana/grafana-image-renderer/issues/676

load docker

function teardown() {
    _remove_docker
}

@test "docker image contains openssl tools" {
    # The openssl tools are required to convert from PEM to CRT format.
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'openssl version'
    [ "$status" -eq 0 ]
    [ -n "$output" ]
}

@test "docker image contains update-ca-certificates" {
    # We need a way to regenerate the CA certificate bundle.
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'command -v update-ca-certificates'
    [ "$status" -eq 0 ]
    [ -n "$output" ]
}

@test "docker image contains certutil" {
    # certutil is required to load the CA certificate bundle into NSS, which is used by Chromium.
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'command -v certutil'
    [ "$status" -eq 0 ]
    [ -n "$output" ]
}
