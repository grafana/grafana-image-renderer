# Nix image

## Build and test

From the repository root, run:

```sh
make image-nix
IMAGE=image-renderer-nix go test ./tests/acceptance/... -count=1
```

Docker with BuildKit runs Nix inside a Linux builder. On macOS, start Docker
Desktop or OrbStack first. The build loads the image as `image-renderer-nix`
and saves the archive to `dist/nix/<system>/image.tar.gz`. Use `NIX_IMAGE` to
change the tag or `NIX_SYSTEM=x86_64-linux` or `aarch64-linux` to select the
platform. Building for a different architecture requires Docker emulation.
The Docker builder uses Docker's isolation with Nix's nested sandbox and
syscall filtering disabled. On native Linux, use `nix build .#image` directly.
For private CAs, see the [container configuration](../docs/sources/troubleshooting.md#nix-container).

[flake.lock](../flake.lock) pins Nixpkgs, including Go and Chromium. The image
uses the upstream browser when the override is empty, equal, or older. A newer
[chromium-info.json](chromium-info.json) selects the Nixpkgs Chromium recipe
with the recorded source revisions and hashes. Nix uses a cached browser when
available. Compiling it on a cache miss can take several hours.

Docker Tests runs acceptance tests on amd64 and arm64. The Build workflow
independently builds and publishes the release images as `<version>-nix` and
`latest-nix`. With an uncached manual override, both workflows compile
Chromium on each architecture.

## Update Chromium

Renovate owns routine updates to `flake.lock`. Each lock-file PR builds and tests
the renderer with the Chromium selected by that revision.

If a Linux stable fix is needed before Nixpkgs packages it, generate its source
revisions and hashes with the updater from the **pinned Nixpkgs revision**.
From the renderer repository root on Linux with Nix, Git, and jq, run:

```sh
nix eval --raw .#browser.version  # currently selected version
rev=$(jq -r '.nodes.nixpkgs.locked.rev' flake.lock)
work=$(mktemp -d)
(
  set -e
  cd "$work"
  git init -q
  git fetch --quiet --depth=1 https://github.com/NixOS/nixpkgs.git "$rev"
  git checkout --quiet --detach FETCH_HEAD
  recipe=pkgs/applications/networking/browsers/chromium
  NIX_PATH="nixpkgs=$PWD" "$recipe/update.mjs" --chromium
  jq -S '{chromium}' "$recipe/info.json" > candidate.json
)
jq -r '.chromium.version' "$work/candidate.json"
```

If the candidate contains the newer release you need, copy the complete entry
and verify selection before opening a PR:

```sh
cp "$work/candidate.json" nix/chromium-info.json
nix eval --raw .#browser.version
nix flake check --all-systems --no-build
git diff -- nix/chromium-info.json
```

Commit only `nix/chromium-info.json` for the override. CI builds and tests it
on both architectures. When a Renovate pin catches up, selection automatically
returns to Nixpkgs. Reset `nix/chromium-info.json` to `{}` in that PR to drop
the stale metadata.

## Update the Go builder

Keep the Go builder in [renderer.nix](renderer.nix) aligned with `go.mod`.
After Go dependency changes, use `pkgs.lib.fakeHash` for `vendorHash`, build
once, and replace it with the hash reported by Nix.
