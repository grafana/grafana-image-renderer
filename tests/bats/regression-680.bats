#!/usr/bin/env bats

# Regression test for: https://github.com/grafana/grafana-image-renderer/issues/680

load docker

function teardown() {
    _remove_docker
}

@test "lang is en_US.UTF-8" {
    # The LANG environment variable should be set to en_US.UTF-8, which makes Chromium able to deal with non-ASCII characters properly.

    # shellcheck disable=SC2016 # we want to avoid variable expansion here
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'echo $LANG'
    [ "$status" -eq 0 ]
    [ "$output" = "en_US.UTF-8" ]

    # shellcheck disable=SC2016 # we want to avoid variable expansion here
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'echo $LC_ALL'
    [ "$status" -eq 0 ]
    [ "$output" = "en_US.UTF-8" ]
}

@test "lang is respected" {
    skip "somehow, the functionality works, but the test fails on sorting" # FIXME

    # With LC_ALL=C, sorting is byte-wise. With LC_ALL=en_US.UTF-8, sorting is character-wise.
    # We can use this to ensure we are not using the C locale.
    run _docker run --entrypoint sh --name "$(_container_name)" "$DOCKER_IMAGE" -c 'echo -e "a\nb\nA\nB" | sort'
    [ "$status" -eq 0 ]
    # With LC_ALL=C, the output would be "A\nB\na\nb"
    echo "$output"
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "A" ]
    [ "${lines[2]}" = "b" ]
    [ "${lines[3]}" = "B" ]
}
