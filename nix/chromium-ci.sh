#!/usr/bin/env bash

chromium_versions() {
  local ref
  ref=$(jq -er --arg input "$1" '
    .nodes.nixpkgs[$input] | select(.type == "github")
    | "github:\(.owner)/\(.repo)/\(.rev // .ref)"
  ' flake.lock) || return
  nix eval --json "$ref#legacyPackages" --apply \
    'pkgs: [ pkgs.x86_64-linux.chromium.browser.version pkgs.aarch64-linux.chromium.browser.version ]'
}

custom_chromium_version() {
  jq -er '.chromium.version | select(test("^[0-9]+(\\.[0-9]+){3}$"))' nix/chromium-info.json
}

chromium_version_is_newer() {
  [[ "$1" != "$2" && $(printf '%s\n' "$1" "$2" | sort -V | tail -n 1) == "$1" ]]
}
