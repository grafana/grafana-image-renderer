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
  
Open a PR, merge it, and create a new GitHub release. You're done!

## Updating Go

Renovate updates the `go` directive and the pinned Alpine builder image together once the matching Go 1.x image is published. It runs `go mod tidy` and includes any resulting changes in the same PR.

**Note**: On minor updates, golangci-lint might break. Also update the CI action to use the latest version, once that is available with support for the new minor.

## Updating Debian

Renovate updates the dated Debian base image tag and its SHA-256 digest in the `Dockerfile`.
To switch Debian releases, update the codename and pin the new image digest manually.
Changing the base image invalidates the package cache. To refresh packages between image updates, update the cachebuster date.

To update it manually:
1. Update the image in the `Dockerfile` starting with `FROM debian:...` with the new version. Don't forget to pin it with the sha256!
    - You can find it by going to DockerHub, or pulling the image yourself you can see a `Digest: ...` line with the value.
2. Bust the cache. Changing the base image will invalidate it anyways but for good manners set it to today's date.
