---
aliases:
  - ../administration/image_rendering/
  - ../image-rendering/
description: Image rendering
keywords:
  - grafana
  - image
  - rendering
  - plugin
labels:
  products:
    - enterprise
    - oss
title: Set up image rendering
weight: 1000
---

# Set up image rendering

The [Grafana Image Renderer service][github] enables you to render web pages in
Grafana to PDFs, PNGs, and CSVs. For example, you can export your panels or
dashboards to PDFs to share with business stakeholders, or set up automatic
exports for Slack alert notifications, so you see your failing alerts with
context. Some features are exclusive to Grafana Enterprise and Grafana Cloud.

{{< admonition type="caution" >}}
We historically provided this functionality as a Grafana plugin. This plugin is
deprecated and does not receive updates anymore. Similarly, we only support the
latest version of the service.
{{< /admonition >}}

{{< admonition type="note" >}}
At any given time, we commit to supporting _all_ stable currently supported
versions of Grafana. If you find a bug, consider reporting it on our [issue
tracker][issues].

[issues]: https://github.com/grafana/grafana-image-renderer/issues

{{< /admonition >}}

## Installation

To install the service, we recommend using Docker or other containerisation
software. This is not strictly required, however, so you can adapt to your
specific environment as needed.

We [ship images][image] for `linux/amd64` and `linux/arm64`; Windows and macOS
can run these images with Docker Desktop. Additionally, we ship binaries for
Windows and Linux on our [GitHub Releases page][releases].

To run the service, use a server with significant memory and CPU resources
available. Some rendering tasks can require several gigabytes of memory and many
cores in a spiky pattern. For a general recommendation, we suggest you allocate
at least 16 GiB of memory and at least 4 CPU cores; you may have to adapt this
to your specific use-case if your load is significant.

### Install with Docker

While we ship a `latest` tag, you should generally prefer pinning a specific
version in production environments. We do, however, commit to keeping the
`latest` tag the latest stable release.

```shell
docker network create grafana
docker run --network grafana --name renderer --rm --detach grafana/grafana-image-renderer:latest
# The following is not a production-ready Grafana instance, but shows what env vars you should set:
docker run --network grafana --name grafana --rm --detach --env GF_RENDERING_SERVER_URL=http://renderer:8081/render --env http://grafana:3000/ --port 3000:3000 grafana/grafana-enterprise:latest
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

If you are running with memory limits, you may want to set the environment
variable `GOMEMLIMIT` to a lower value than the limit, such as `1GiB`. You
should not aim for the `GOMEMLIMIT` to match the container's limit because
Chromium needs free memory on top. We recommend 1 GiB of `GOMEMLIMIT` per 8 GiB
of container memory limit.

If you do not use memory limits, the previous configuration can still be useful,
especially if you run the service alongside other services on the same host.

### Install with binaries

Binaries are released for Linux and Windows, for `amd64` and `arm64`
architectures. macOS is not supported at this time; you should prefer Docker
Desktop.

The binaries do not ship with a browser; you must install your own
Chromium-based browser on your own, and configure it to be used. Examples of
these include [Google Chrome](https://www.google.com/chrome/) and
[Microsoft Edge](https://www.microsoft.com/en-us/edge). Configure the path to
this browser binary with `--browser.path`.

The binaries are hosted on [GitHub Releases][releases]. Download the appropriate
binary for your platform, and run it from the terminal. You would run something
like `./grafana-image-renderer server`; see `--help` for more information.

### Grafana configuration

Grafana includes options to specify where to find the renderer service, and
which authentication token to use in communication. These options are:

```ini
[rendering]
; The URL is an HTTP(S) URL to the service, and its rendering endpoint.
; By default, the path is `/render`. This can be changed with nginx or similar reverse proxies.
server_url = http://renderer:8081/render

; The token is any _one_ token configured in the renderer service.
; If you configured multiple tokens in the service, choose one for Grafana.
; By default, a single token is configured, with the value of `-`.
renderer_token = -
```

With Docker or similar setups that use environment variables, the options are:

- `GF_RENDERING_SERVER_URL`, and
- `GF_RENDERING_RENDERER_TOKEN`.

## Configuration

You can configure the service several ways:

- Set CLI flags. See `--help` for the available flag names.
- Use environment variables. See `--help` for the names and current values.
  Most, but not all, variables map 1:1 to the CLI flag names.
- Use a JSON or YAML configuration file. This must be in the service's current
  working directory, and must be named either `config.json`, `config.yaml`, or
  `config.yml`. Dot-separated keys are nested keys. For example: `a.b` becomes
  `{"a": {"b": "VALUE"}}` in the file.

You can see all the current configuration options by checking `--help`.

For example, see: `docker run --rm grafana/grafana-image-renderer:latest
server --help`.

### Security

The service requires a secret token to be present in all render requests. This
token is set by `--server.auth-token` (`AUTH_TOKEN`); you can specify multiple
and have a unique key per Grafana instance. By default, the token used is `-`.
In Grafana, you must set this to match one of the tokens with the `[rendering]
renderer_token` (`GF_RENDERING_RENDERER_TOKEN`) configuration setting.

## Monitoring

You can monitor the service via Prometheus or
[Mimir](https://grafana.com/oss/mimir) and any OpenTelemetry-compatible Tracing
backend (like [Grafana Tempo](https://grafana.com/oss/tempo)). Metrics are
exposed on `/metrics` and traces are sent as configured in the configuration
options (see `--help`).

You can find an [example dashboard here](https://grafana.com/grafana/dashboards/12203-grafana-image-renderer/).

[github]: https://github.com/grafana/grafana-image-renderer
[image]: https://hub.docker.com/r/grafana/grafana-image-renderer
[releases]: https://github.com/grafana/grafana-image-renderer/releases
