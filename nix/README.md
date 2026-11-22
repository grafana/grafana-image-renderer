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

[flake.lock](../flake.lock) pins Nixpkgs, including Go and Chromium. The image
uses the upstream browser when the override is empty, equal, or older. A newer
[chromium-info.json](chromium-info.json) selects the Nixpkgs Chromium recipe
with the recorded source revisions and hashes. Nix uses a cached browser when
available. Compiling it on a cache miss can take several hours.

For an override, CI caches the browser and sandbox runtime closure under the exact
derivation and architecture. GitHub cache scope and eviction determine which
runs can reuse it. Each architecture runs acceptance tests against the built
archive. Publication loads that archive to produce the `<version>-nix` and
`latest-nix` tags.

## Update Chromium

The [daily workflow](../.github/workflows/update-nix-chromium.yaml) checks the
Linux stable release. It updates the Nixpkgs lock when the channel already
packages the release. Otherwise it runs the pinned Nixpkgs updater to record
the newer Chromium sources. Once the pin catches up, the workflow clears the
override to `{}`. Pull requests and release tags use the same selection.

To refresh source metadata locally, run `make update-nix-chromium`. This checks
the live release even on an unchanged checkout and fetches the required sources
without compiling the browser. Review changes to `chromium-info.json` before
committing. Run `nix flake update nixpkgs` to refresh the whole package set.

Keep the Go builder in [renderer.nix](renderer.nix) aligned with `go.mod`.
After Go dependency changes, use `pkgs.lib.fakeHash` for `vendorHash`, build
once, and replace it with the hash reported by Nix. For private CAs, see the
[container configuration](../docs/sources/troubleshooting.md#nix-container).
