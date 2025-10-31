# grafana-image-renderer

A backend service for Grafana.
It provides panel and dashboard rendering with a headless browser (Chromium).
You can get your favourite dashboards as PDFs, PNGs, or with Grafana Enterprise, CSVs and even over emails with Grafana Reports.

## Installation

To install the service, you are recommended to use Docker or other containerisation software.
We [ship images][image] for `linux/amd64` and `linux/arm64`; Windows and macOS can run these with Docker Desktop as well.
We do not currently ship a binary you can run, nor do we ship a plugin.

To run the service in a stable manner, you are recommended to have a minimum of 16 GiB memory allocated to this service.
You may also require a relatively modern CPU for decent output speeds.

### Install with Docker

While we ship a `latest` tag, you should generally prefer pinning a specific version in production environments.
We do, however, commit to keeping the `latest` tag the latest stable release.

```shell
$ docker network create grafana
$ docker run --network grafana --name renderer --rm --detach grafana/grafana-image-renderer:latest
# The following is not a production-ready Grafana instance, but shows what env vars you should set:
$ docker run --network grafana --name grafana --rm --detach --env GF_RENDERING_SERVER_URL=http://renderer:8081/render --env http://grafana:3000/ --port 3000:3000 grafana/grafana-enterprise:latest
```

Alternatively, if you prefer `docker compose`:

```yaml
services:
  renderer:
    image: grafana/grafana-image-renderer:latest

  grafana:
    image: grafana/grafana-enterprise:latest
    ports:
      - "3000:3000"
    environment:
      GF_RENDERING_SERVER_URL: http://renderer:8081/render
      GF_RENDERING_CALLBACK_URL: http://grafana:3000/
```

If you are running with memory limits, you may want to set `GOMEMLIMIT` to a lower value than the limit, such as `1GiB`.
You should not aim for the `GOMEMLIMIT` to match the container's limit: Chromium needs free memory on top.
We recommend 1 GiB of `GOMEMLIMIT` per 8 GiB of container memory limit.

[image]: https://hub.docker.com/r/grafana/grafana-image-renderer

### Configuration

The service can be configured via several paths:

- Set CLI flags. See `--help` for the available flag names.
- Use environment variables. See `--help` for the names and current values.
  Most, but not all, variables map 1:1 to the CLI flag names.
- Use a JSON or YAML configuration file. This must be in the service's current working directory,
  and must be named one of `config.json`, `config.yaml`, or `config.yml`. Dot-separated keys are
  nested keys. E.g.: `a.b` becomes `{"a": {"b": "VALUE"}}` in the file.

The current configuration options can all be accessed by checking `--help`.

### Security

The service requires a secret token to be present in all render requests.
This token is set by `--server.auth-token` (`AUTH_TOKEN`); you can specify multiple and have a unique key per Grafana instance.
By default, the token used is `-`.
In Grafana, you must set this to match one of the tokens with the `[rendering] renderer_token` (`GF_RENDERING_RENDERER_TOKEN`) setting.

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
