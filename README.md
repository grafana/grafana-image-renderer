# grafana-image-renderer

A backend service for Grafana.
It provides panel and dashboard rendering with a headless browser (Chromium).
You can get your favourite dashboards as PDFs, PNGs, or with Grafana Enterprise, CSVs and even over emails with Grafana Reports.

## Installation

You can find installation details in the [docs](/docs/sources/_index.md).

<!-- FIXME: Use the grafana.com docs when it's published -->

## Compile

To compile the Go service, run:

```shell
$ go build -buildvcs -o grafana-image-renderer .
```

To compile the Docker image, run:

```shell
$ docker build .
```

An additional Nix image uses Nixpkgs Chromium and builds the Go service with
`buildGoModule`. It launches the unwrapped browser with the renderer's font,
locale and sandbox configuration, keeping desktop integration tools out of the
runtime closure. Both `x86_64-linux` and `aarch64-linux` are supported:

```shell
$ make image-nix # loads image-renderer-nix into Docker
$ IMAGE=image-renderer-nix go test ./tests/acceptance/... -count=1
```

By default, the image uses upstream Nixpkgs Chromium, allowing Nix to reuse its
binary cache. To build Chromium with CUPS support disabled instead:

```shell
$ make image-nix NIX_CHROMIUM=custom
```

The custom variant compiles Chromium from source on the first build for each
architecture. The measured ARM64 build took almost six hours with eight compiler
jobs. When invoking Nix directly, select `.#packages.<system>.image-custom`.
[Chromium's build requirements](https://chromium.googlesource.com/chromium/src/+/main/docs/linux/build_instructions.md)
call for at least 100 GB of free disk space.

To select an upstream release before Nixpkgs packages it, first generate its
source metadata, then build the custom variant:

```shell
$ make update-nix-chromium # latest stable Linux release
$ make image-nix NIX_CHROMIUM=custom
```

The update target runs the unmodified Nixpkgs Chromium updater inside Docker.
It uses its Linux stable-channel release selection and updates only when a
newer release is available. It resolves dependency revisions and source hashes into
`nix/chromium-info.json`; it fetches sources without compiling Chromium. That
file is committed to pin the custom version independently of `flake.lock`.
Upstream mode always uses the unmodified Nixpkgs browser. Unchanged metadata
preserves the Chromium derivation and reuses its result when available in the
local Nix store or a binary cache. The existing Nixpkgs build recipe and patch
rules apply to custom releases too; newer releases can require patch or
toolchain updates.

Local recipe fixes can be listed, in application order, in
`nix/chromium-recipe-patches.nix`:

```nix
[
  ./chromium-next-major.patch
]
```

Patch paths are relative to Nixpkgs' Chromium recipe directory: use headers such
as `a/common.nix`, `b/common.nix` or `a/update.mjs`, `b/update.mjs`. The shared
`nix/chromium-recipe.nix` prepares that patched directory for both the custom
browser and metadata updater. Keep release data in `nix/chromium-info.json`.
With no recipe patches, the existing custom Chromium derivation is preserved.
A nonempty list requires a small source-preparation derivation before the
patched recipe is evaluated; it does not compile Chromium itself. Recipe changes can invalidate the browser
cache. The upstream image variant continues to use the unmodified package.

This requires Docker with BuildKit; Nix runs inside the Docker build. On macOS,
start Docker Desktop or OrbStack first. No host Nix installation, FlakeHub
account, or remote Nix builder is needed. Docker keeps the Nix store in a build
cache per architecture and exports the image archive to
`dist/nix/<system>/image.tar.gz` before loading it.

Override `NIX_SYSTEM` to select the other architecture, and `NIX_IMAGE` to change
the local tag:

```shell
$ make image-nix NIX_SYSTEM=x86_64-linux NIX_IMAGE=image-renderer-nix-amd64
```

Cross-architecture builds require Docker's emulation support. The build
container uses Docker's isolation with Nix's nested sandbox and syscall filter
disabled, including for emulated builds.

CI builds both architectures natively and publishes the additional
`<version>-nix` and `latest-nix` tags.

`flake.lock` pins Nixpkgs, including Chromium and Go. Update it with
`nix flake update nixpkgs`, then test both architectures. Keep the versioned Go
builder in `nix/renderer.nix` aligned with `go.mod`. When Go dependencies change,
set `vendorHash` to `pkgs.lib.fakeHash`, build once, and replace it with the hash
reported by Nix. Docker builds the working tree, filtered by
`Dockerfile.nix.dockerignore`. When invoking Nix directly, build from the Git
flake (`.#packages.aarch64-linux.image`); new source files must be added to Git
to be included.

The image uses an immutable CA bundle from `dockerTools.caCertificates`.
See [custom CA certificates](docs/sources/troubleshooting.md#nix-container) for
Nix-specific configuration.

The following tools are also useful for engineers:

```shell
$ go tool goimports # our code formatter of choice
$ golangci-lint run # our linter of choice
$ IMAGE=tag-here go test ./tests/acceptance/... -count=1 # run acceptance tests
```

## Release

When you are ready to make a release, tag it in Git (`git tag vX.Y.Z`; remember the `v` prefix), then push it.
GitHub Actions deals with the build and deployment.

## Updating Chromium

One of the quick ways you can check the latest version available is by going to [https://packages.debian.org/trixie/chromium](https://packages.debian.org/trixie/chromium) (change `trixie` for the codename of the current Debian version we use in the `Dockerfile`, as of writing `13`).
    
Make sure that the build for amd64 and arm64 are both available first (scroll down to the bottom of the page and check that there are no red backgrounds), and ignore anything after the `-` in the version, this is what we'll use.
  
In the `Dockerfile`, update `CHROMIUM_VERSION=` to the version found on the website. Changing this argument also reruns the package update and upgrade step. Use the cachebuster date for manual package refreshes.

For the custom Nix variant, refresh the upstream release metadata with
`make update-nix-chromium` and include the generated `nix/chromium-info.json` in
the update PR. This can select a newer Linux stable release ahead of Nixpkgs
without changing `flake.lock`. If no newer release is available, the metadata
stays unchanged. Build with `NIX_CHROMIUM=custom` and run the
acceptance tests when validating the update. The default Nix variant gets its
Chromium version through `nix flake update nixpkgs`.

Open a PR, merge it, and create a new GitHub release. You're done!

## Updating Go

For a new minor N, only update to a `.0` patch if there are security vulnerabilities that are fixed and not present in the N-1 minor. Otherwise try to stick with the N-1 minor until the N minor is at least on a `.1` patch.
  
1. Change the `go` directive to the new version in `go.mod`. 
2. Update the image at the top of the `Dockerfile` starting with `FROM golang:...` with the new version. Don't forget to pin it with the sha256!
    - You can find it by going to DockerHub, or pulling the image yourself you can see a `Digest: ...` line with the value. 

**Note**: On minor updates, golangci-lint might break. Also update the CI action to use the latest version, once that is available with support for the new minor.

## Updating Debian

It is good to update whenever a new image is tagged, as it mainly contains package updates or potential security fixes which are always nice to have.
  
1. Update the image in the `Dockerfile` starting with `FROM debian:...` with the new version. Don't forget to pin it with the sha256!
    - You can find it by going to DockerHub, or pulling the image yourself you can see a `Digest: ...` line with the value.
2. Bust the cache. Changing the base image will invalidate it anyways but for good manners set it to today's date.
