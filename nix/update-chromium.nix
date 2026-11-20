{ pkgs }:

let
  recipe = import ./chromium-recipe.nix { inherit pkgs; };
in
pkgs.writeShellApplication {
  name = "update-chromium";
  runtimeInputs = [
    pkgs.coreutils
    pkgs.git
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
    chmod u+w "$work/nixpkgs/$(dirname "$chromium_dir")" "$work/nixpkgs/$chromium_dir"
    mv "$work/nixpkgs/$chromium_dir" "$work/original-chromium"
    cp -R ${recipe.source} "$work/nixpkgs/$chromium_dir"
    chmod -R u+w "$work/nixpkgs"
    cd "$work/nixpkgs"
    git init --quiet

    cp ${./chromium-info.json} "$chromium_dir/info.json"
    export NIX_PATH="nixpkgs=$PWD"
    zx "$chromium_dir/update.mjs" --chromium
    cp "$chromium_dir/info.json" "$output"
  '';
}
