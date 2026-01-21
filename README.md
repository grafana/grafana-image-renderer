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
  
In the `Dockerfile`, update the cachebuster date to today's date. Then find `CHROMIUM_VERSION=` and update that to the version found in the website.
  
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
