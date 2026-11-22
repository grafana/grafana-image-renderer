#!/usr/bin/env bash
set -euo pipefail

nixpkgs_chromium_version() {
  local ref
  ref=$(jq -er --arg input "$1" '
    .nodes.nixpkgs[$input] | select(.type == "github")
    | "github:\(.owner)/\(.repo)/\(.rev // .ref)"
  ' flake.lock) || return
  nix eval --raw "$ref#legacyPackages.x86_64-linux.chromium.version"
}

chromium_version_is_newer() {
  [[ "$1" != "$2" && $(printf '%s\n' "$1" "$2" | sort -V | tail -n 1) == "$1" ]]
}

# Use the same rollout threshold as the Nixpkgs Chromium updater.
latest=$(curl --fail --silent --show-error --retry 3 --max-time 30 --get \
  --data-urlencode 'order_by=version desc' \
  --data-urlencode 'filter=endtime=none,fraction>=0.5' \
  https://versionhistory.googleapis.com/v1/chrome/platforms/linux/channels/stable/versions/all/releases \
  | jq -er '.releases[0].version | select(test("^[0-9]+(\\.[0-9]+){3}$"))')

pinned=$(nixpkgs_chromium_version locked)
if chromium_version_is_newer "$latest" "$pinned"; then
  available=$(nixpkgs_chromium_version original)
  if ! chromium_version_is_newer "$latest" "$available"; then
    nix flake update nixpkgs
    pinned=$(nixpkgs_chromium_version locked)
    if chromium_version_is_newer "$latest" "$pinned"; then
      echo "Updated pin has Chromium $pinned, expected at least $latest" >&2
      exit 1
    fi
  else
    selected=$(nix eval --raw .#browser.version)
    if chromium_version_is_newer "$latest" "$selected"; then
      nix run --no-update-lock-file .#update-chromium -- nix/chromium-info.json
      selected=$(nix eval --raw .#browser.version)
      if chromium_version_is_newer "$latest" "$selected"; then
        echo "Updater selected Chromium $selected, expected at least $latest" >&2
        exit 1
      fi
    fi
  fi
fi

override=$(jq -r '.chromium.version // empty' nix/chromium-info.json)
if [[ -n "$override" ]] && ! chromium_version_is_newer "$override" "$pinned"; then
  printf '{}\n' > nix/chromium-info.json
fi
if ! git diff --quiet -- flake.lock nix/chromium-info.json; then
  selected=$(nix eval --raw .#browser.version)
  echo "title=Update Nix Chromium to $selected" >> "$GITHUB_OUTPUT"
fi
