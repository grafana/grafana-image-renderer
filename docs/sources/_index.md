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

The [Grafana Image Renderer service][github] enables you to render Grafana panels and dashboards as PNGs, PDFs, or CSV files.

You can export visualizations to share with stakeholders or configure automatic exports for alert notifications.

Image rendering works with Grafana OSS, Grafana Enterprise, and Grafana Cloud.
Some features are exclusive to Grafana Enterprise and Grafana Cloud.

{{< admonition type="caution" >}}
We historically provided this functionality as a Grafana plugin.
This plugin is deprecated and no longer receives updates.
Similarly, we only support the latest version of the service.
{{< /admonition >}}

{{< admonition type="note" >}}
At any given time, we commit to supporting _all_ stable currently supported versions of Grafana.
If you find a bug, consider reporting it on our [issue tracker][issues].

[issues]: https://github.com/grafana/grafana-image-renderer/issues
{{< /admonition >}}

## Before you begin

Before you install the image renderer, ensure you have the following:

- A running Grafana instance
- Docker installed, or the ability to run binaries on Linux or Windows
- At least 16 GiB of memory and 4 CPU cores available for the renderer service

## Installation

To install the service, we recommend using Docker or other containerization software.
This isn't strictly required, so you can adapt to your specific environment as needed.

We [ship images][image] for `linux/amd64` and `linux/arm64`.
Windows and macOS can run these images with Docker Desktop.
Additionally, we ship binaries for Windows and Linux on our [GitHub Releases page][releases].

To run the service, use a server with significant memory and CPU resources available.
Some rendering tasks require several gigabytes of memory and many cores in a spiky pattern.
We recommend you allocate at least 16 GiB of memory and at least 4 CPU cores.
You may need to adapt this to your specific use case if your load is significant.

### Install with Docker

While we ship a `latest` tag, prefer pinning a specific version in production environments.
We commit to keeping the `latest` tag the latest stable release.

The following example creates a Docker network and starts both the renderer and Grafana services.
This configuration demonstrates what environment variables to set, but it doesn't use a production-ready Grafana instance:

```shell
docker network create grafana
docker run --network grafana --name renderer --rm --detach grafana/grafana-image-renderer:latest
docker run --network grafana --name grafana --rm --detach --env GF_RENDERING_SERVER_URL=http://renderer:8081/render --env GF_RENDERING_CALLBACK_URL=http://grafana:3000/ --port 3000:3000 grafana/grafana-enterprise:latest
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

If you're running with memory limits, set the `GOMEMLIMIT` environment variable to a lower value than the limit, such as `1GiB`.
Don't set `GOMEMLIMIT` to match the container's limit because Chromium needs free memory on top.
We recommend 1 GiB of `GOMEMLIMIT` per 8 GiB of container memory limit.

If you don't use memory limits, the previous configuration can still be useful, especially if you run the service alongside other services on the same host.

### Install with binaries

Binaries are released for Linux and Windows, for `amd64` and `arm64` architectures.
macOS isn't supported at this time.
If you're using macOS, prefer Docker Desktop.

The binaries don't ship with a browser.
You must install your own Chromium-based browser and configure it to be used.
Examples include [Google Chrome](https://www.google.com/chrome/) and [Microsoft Edge](https://www.microsoft.com/en-us/edge).
Configure the path to this browser binary with `--browser.path`.

The binaries are hosted on [GitHub Releases][releases].
Download the appropriate binary for your platform and run it from the terminal.
For example, run `./grafana-image-renderer server`.
See `--help` for more information.

### Grafana configuration

Grafana includes options to specify where to find the renderer service and which authentication token to use in communication.

To configure these, use the following options in the `.ini` configuration file for Grafana:

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

For Docker or similar setups that use environment variables, use the following options:

- `GF_RENDERING_SERVER_URL`
- `GF_RENDERING_RENDERER_TOKEN`

## Configuration

You can configure the service in the following ways:

- **CLI flags**: Run with `--help` to see available flag names.
- **Environment variables**: Run with `--help` to see names and current values.
  Most, but not all, variables map 1:1 to the CLI flag names.
- **Configuration file**: Create a JSON or YAML configuration file in the service's working directory.
  Name the file `config.json`, `config.yaml`, or `config.yml`.
  Dot-separated keys are nested keys.
  For example, `a.b` becomes `{"a": {"b": "VALUE"}}` in the file.

You can use `--help` to see all configuration options.
For example, if you're using Docker:

```shell
docker run --rm grafana/grafana-image-renderer:latest server --help
```

### Security

The service requires a secret token to be present in all render requests.
Set this token using `--server.auth-token` (`AUTH_TOKEN`).
You can specify multiple tokens and have a unique key per Grafana instance.
By default, the token used is `-`.
In Grafana, set this to match one of the tokens with the `[rendering] renderer_token` (`GF_RENDERING_RENDERER_TOKEN`) configuration setting.

## Monitor the image renderer

You can monitor the service using Prometheus or [Grafana Mimir](https://grafana.com/oss/mimir) for metrics and any OpenTelemetry-compatible tracing backend for traces, such as [Grafana Tempo](https://grafana.com/oss/tempo).

Metrics are exposed on the `/metrics` endpoint.
Traces are sent as configured in the configuration options.
See `--help` for trace configuration options.

For a pre-built monitoring dashboard, refer to this [example dashboard](https://grafana.com/grafana/dashboards/12203-grafana-image-renderer/).

[github]: https://github.com/grafana/grafana-image-renderer
[image]: https://hub.docker.com/r/grafana/grafana-image-renderer
[releases]: https://github.com/grafana/grafana-image-renderer/releases
