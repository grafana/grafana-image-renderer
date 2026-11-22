# Nix image

These files build the Grafana Image Renderer container image for `amd64` and
`arm64` Linux. [flake.nix](../flake.nix) defines the image and update targets.
[flake.lock](../flake.lock) pins the Nixpkgs revision that supplies the build
tools and packages.

## Files

| File | Purpose |
| --- | --- |
| [Makefile](Makefile) | Defines the image build and Chromium update commands. The root Makefile includes this file. |
| [Dockerfile.nix](Dockerfile.nix) | Runs the image build and the Chromium updater inside Docker. |
| [Dockerfile.nix.dockerignore](Dockerfile.nix.dockerignore) | Lists files to exclude from the Docker build context. The context starts at the repository root. |
| [renderer.nix](renderer.nix) | Builds the renderer with Go and runs the short test suite. The `vendorHash` field checks the contents of the vendored Go dependencies. |
| [image.nix](image.nix) | Puts the renderer and Chromium in a Docker image. It includes fonts and CA certificates. It also sets the runtime user, Chromium sandbox, and health check. |
| [chromium.nix](chromium.nix) | Selects Chromium for the custom image. It uses the Nixpkgs recipe with the release data from `chromium-info.json`. |
| [chromium-info.json](chromium-info.json) | Pins the Chromium release for the custom image. It records the source revisions and hashes for Chromium and its dependencies. |
| [chromium-ci.sh](chromium-ci.sh) | Reads and compares Chromium versions for the GitHub workflows. |
| [chromium-recipe.nix](chromium-recipe.nix) | Selects the upstream Chromium recipe and applies any local recipe patches. The custom browser and the updater use the same recipe. |
| [update-chromium.nix](update-chromium.nix) | Runs the Chromium updater in a temporary copy of Nixpkgs. The updater checks the Linux stable channel and writes the release data to `chromium-info.json`. |

## Build an image

Run these commands from the repository root. The [Makefile](../Makefile) loads
the image into Docker with the tag `image-renderer-nix` by default.

```sh
make image-nix
```

This command uses the Chromium package from the Nixpkgs revision in `flake.lock`.
To use the release data in `chromium-info.json`, select the custom image.

```sh
make image-nix NIX_CHROMIUM=custom
```

Nix can use a cached Chromium build if it matches the recipe, release data, and
target architecture. A source build can take several hours.

## Update Chromium

The [daily workflow](../.github/workflows/update-nix-chromium.yaml) opens a pull
request to update `flake.lock` when the Nixpkgs channel has the newer stable
Linux release for both architectures. If the channel is behind, the workflow
updates `chromium-info.json` instead. It can update the Nixpkgs pin when the
channel catches up. The repository must allow GitHub Actions to create pull
requests. GitHub can require approval to run CI on these pull requests.

On pushes to the default branch, the [build workflow](../.github/workflows/build.yaml)
uses custom Chromium if it is newer than the pinned Chromium on both
architectures. Custom builds use `ubuntu-amd64-6xlarge` for `amd64` and
`ubuntu-arm64-xlarge` for `arm64`.

Renovate also checks `flake.lock` weekly and opens a pull request when it can
update the Nixpkgs pin. To update the pin manually, run `nix flake update nixpkgs`.

For the custom image, run `make update-nix-chromium`. This command checks for a
newer Linux stable release and fetches the sources it needs to calculate hashes.
It generates metadata without compiling the browser.

Review the changes to `chromium-info.json`. Build the custom image and run the
acceptance tests before you submit the update.

## Change the Chromium recipe

Add local recipe patches to the `patches` list in `chromium-recipe.nix`. Use
paths relative to the upstream recipe directory in each patch. For example, use
`a/common.nix` and `b/common.nix` to patch `common.nix`.
[Nixpkgs Chromium recipe](https://github.com/NixOS/nixpkgs/tree/master/pkgs/applications/networking/browsers/chromium)
