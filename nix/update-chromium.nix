{ pkgs }:

let
  chromium = import ./chromium.nix { inherit pkgs; };
  seed = pkgs.writeText "chromium-info.json" (builtins.toJSON { chromium = chromium.upstream-info; });
in
pkgs.writeShellApplication {
  name = "update-chromium";
  runtimeInputs = [
    pkgs.coreutils
    pkgs.git
    pkgs.jq
    pkgs.nix
    pkgs.python3
    pkgs.zx
  ];
  text = ''
    if [[ $# != 1 ]]; then
      echo 'Usage: update-chromium OUTPUT.json' >&2
      exit 1
    fi

    output="$(realpath -m "$1")"

    # The upstream updater evaluates packages from a writable Nixpkgs Git root.
    work="$(mktemp -d)"
    cp -R ${pkgs.path}/. "$work/nixpkgs"
    chromium_dir=pkgs/applications/networking/browsers/chromium
    chmod u+w "$work/nixpkgs" "$work/nixpkgs/$chromium_dir" "$work/nixpkgs/$chromium_dir/info.json"
    cd "$work/nixpkgs"
    git init --quiet

    cp ${seed} "$chromium_dir/info.json"
    export NIX_PATH="nixpkgs=$PWD"
    zx "$chromium_dir/update.mjs" --chromium
    jq -S --arg upstream ${pkgs.lib.escapeShellArg pkgs.chromium.version} \
      'if .chromium.version == $upstream then {} else {chromium} end' "$chromium_dir/info.json" > "$output"
  '';
}
