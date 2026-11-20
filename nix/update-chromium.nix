{ pkgs }:

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
    cp -R ${pkgs.path}/. "$work"
    chmod -R u+w "$work"
    cd "$work"
    git init --quiet

    chromium_dir=pkgs/applications/networking/browsers/chromium
    cp ${./chromium-info.json} "$chromium_dir/info.json"
    export NIX_PATH="nixpkgs=${pkgs.path}"
    zx "$chromium_dir/update.mjs" --chromium
    cp "$chromium_dir/info.json" "$output"
  '';
}
