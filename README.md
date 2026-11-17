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

To build the Docker image, run:

```shell
$ make image IMAGE=gir
```

The build uses Docker, melange, and apko to assemble a Wolfi runtime without a shell or package manager.
It installs Debian's precompiled Chromium through signed APT metadata, packages the browser and existing fonts as an APK, and resolves runtime libraries from Wolfi.
The Debian builds of `libjpeg62-turbo` and `libxnvctrl0` supply two library ABIs absent from Wolfi.
It does not compile Chromium.
The Go binary is still built by the root `Dockerfile`.

The melange container requires privileged mode for its build sandbox.
It mounts only the packaging definitions and a temporary build directory.
Packages, a temporary APK signing key, and browser SBOMs remain in that directory; the private key is not included in the image or uploaded by CI.
Set `BUILD_DIR` to choose an output directory, and `ARCH=amd64` or `ARCH=arm64` to select a target (cross builds require Docker's emulation support).
CI builds and tests both architectures on native runners.

To scan an image, install [Syft and Grype](https://oss.anchore.com/docs/installation/) and `jq`, then run:

```shell
$ make scan-image IMAGE=gir
# Pass Grype options directly when needed:
$ sh packaging/scan gir --fail-on high
```

The scan combines the installed Wolfi package inventory with embedded Debian identities for Chromium, its resources and sandbox, and the two Debian libraries.
It checks the complete Debian versions against Debian's security feed and fails if package identities are missing or the browser version is inconsistent.
A plain `grype IMAGE` scan misses these repackaged Debian components because it does not load embedded SBOMs by default.
The script imports only the Debian runtime SBOM; other package SBOMs can also describe build inputs.
The scan exports SPDX JSON so Grype uses each package's distribution instead of applying Wolfi's feed to the Debian components.
Set `SBOM_FILE` to retain this SBOM at a chosen path; otherwise it is saved in a temporary file printed by the scan.

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

The browser asset stage in `packaging/Dockerfile` installs [Debian Chromium](https://packages.debian.org/trixie/chromium).
APT selects the target architecture and verifies the signed repository metadata and package checksums.
No manual SHA256 pins are needed.

Renovate checks Debian's security repository and updates both the upstream version and Debian revision.
For manual updates, check the [Debian package page](https://packages.debian.org/trixie/chromium) and make sure the same version is available for amd64 and arm64.

Update `package.version` in `packaging/browser.yaml` to the upstream browser version (for example, `156.0.8078.79`).
Set `vars.debian-revision` in the same file to Debian's packaging revision (`1~deb13u1` for `156.0.8078.79-1~deb13u1`).
The build passes the complete DEB version to APT for `chromium`, `chromium-common`, and `chromium-sandbox`.
The exact pin makes the build fail if that version is no longer available for the selected architecture.

`packaging/browser.yaml` preserves Chromium's runtime assets and sandbox helper while omitting its desktop shell launcher.
The asset stage retains Debian copyright notices and common license texts for Chromium, the two bundled libraries, and fonts.
The Debian runtime SBOM records the original package and source-package versions.
Font fallback uses Wolfi's fontconfig defaults; Helvetica resolves to Liberation Sans.
Validate changes with `make test-acceptance` on both architectures, including browser sandbox, font, and rendering tests.
  
Open a PR, merge it, and create a new GitHub release. You're done!

## Updating Go

For a new minor N, only update to a `.0` patch if there are security vulnerabilities that are fixed and not present in the N-1 minor. Otherwise try to stick with the N-1 minor until the N minor is at least on a `.1` patch.
  
1. Change the `go` directive to the new version in `go.mod`. 
2. Update the image at the top of the `Dockerfile` starting with `FROM golang:...` with the new version. Don't forget to pin it with the sha256!
    - You can find it by going to DockerHub, or pulling the image yourself you can see a `Digest: ...` line with the value. 

**Note**: On minor updates, golangci-lint might break. Also update the CI action to use the latest version, once that is available with support for the new minor.

## Updating runtime packages and build tools

`packaging/apko.yaml` selects the Wolfi runtime packages.
Each image build resolves their current versions; browser dependencies detected by melange are included automatically.
The image deliberately retains certificate tools, fontconfig, tini, and coreutils (`true` is used by the namespace support probe).

Renovate tracks the melange and apko image digests in `packaging/build`.
Rebuild images regularly to pick up Wolfi package updates.

## Updating the Debian asset stage

It is good to update whenever a new image is tagged, as it mainly contains package updates or potential security fixes which are always nice to have.
  
1. Update the image in `packaging/Dockerfile` starting with `FROM debian:...` with the new version. Don't forget to pin it with the sha256!
    - You can find it by going to DockerHub, or pulling the image yourself you can see a `Digest: ...` line with the value.
2. Rebuild and run the acceptance tests, particularly the font checks and rendering comparisons.
