A Grafana backend plugin that handles rendering panels and dashboards to PNGs using a headless browser (Chromium).

## Requirements

### Supported operating systems

- Linux (x64)
- Windows (x64)
- Mac OS X (x64)

### Dependencies

This plugin is packaged in a single executable with [Node.js](https://nodejs.org/) runtime and [Chromium browser](https://www.chromium.org/Home).
This means that you don't need to have Node.js and Chromium installed in your system for the plugin to function.

However, [Chromium browser](https://www.chromium.org/) depends on certain libraries and if you don't have all of those libraries installed in your
system you may encounter errors when trying to render an image. For further details and troubleshooting help, please refer to
[Grafana Image Rendering documentation](https://grafana.com/docs/administration/image_rendering/).

### Known issue having IPv6 disabled

We've got reports regarding the plugin doesn't work if IPv6 is disabled. The current workaround is to use [Remote Rendering Using Docker](#remote-rendering-using-docker) instead. Refer to [issue](https://github.com/grafana/grafana-image-renderer/issues/48) for more information.

## Installation

### Using grafana-cli

**NOTE:** Installing this plugin using grafana-cli is supported from Grafana v6.4.

```bash
grafana-cli plugins install grafana-image-renderer
```

### Install in Grafana Docker image

This plugin is not compatible with the current Grafana Docker image without installing further system-level dependencies. We recommend setting up another Docker container
for rendering and using remote rendering, see [Remote Rendering Using Docker](#remote-rendering-using-docker) for reference.

If you still want to install the plugin in the Grafana docker image we provide instructions for how to build a custom Grafana image, see [Grafana Docker documentation](https://grafana.com/docs/installation/docker/#custom-image-with-grafana-image-renderer-plugin-pre-installed) for further instructions.

### Environment variables

You can override certain settings by using environment variables and making sure that those are available for the Grafana process.

**Default timezone:**

Instruct headless browser instance to use a default timezone when not provided by Grafana, e.g. when rendering panel image of alert. See [ICU’s metaZones.txt](https://cs.chromium.org/chromium/src/third_party/icu/source/data/misc/metaZones.txt?rcl=faee8bc70570192d82d2978a71e2a615788597d1) for a list of supported timezone IDs. Fallbacks to `TZ` environment variable if not set.

```bash
GF_RENDERER_PLUGIN_TZ=Europe/Stockholm
```

**Ignore HTTPS errors:**

Instruct headless browser instance whether to ignore HTTPS errors during navigation. Per default HTTPS errors is not ignored.
Due to the security risk it's not recommended to ignore HTTPS errors.

```bash
GF_RENDERER_PLUGIN_IGNORE_HTTPS_ERRORS=true
```

**gRPC port:**

Change the listening port of the gRPC server. Default is `0` and will automatically assign a port not in use.

```bash
GF_RENDERER_PLUGIN_GRPC_PORT=50059
```

**Verbose logging:**

Instruct headless browser instance whether to capture and log verbose information when rendering an image. Default is `false` and will only capture and log error messages. When enabled, `true`, debug messages are captured and logged as well.

For the verbose information to be included in the Grafana server log you have to adjust the rendering log level to `debug`, see [Troubleshoot image rendering](https://grafana.com/docs/grafana/latest/administration/image_rendering/#troubleshoot-image-rendering) for instructions.

```bash
GF_RENDERER_PLUGIN_VERBOSE_LOGGING=true
```

## Remote Rendering Using Docker

Instead of installing and running the image renderer as a plugin, you can run it as a remote image rendering service using Docker. Read more about [remote rendering using Docker](https://github.com/grafana/grafana-image-renderer/blob/v1.x/docs/remote_rendering_using_docker.md).

## Troubleshooting

For troubleshooting help, please refer to [Grafana Image Rendering documentation](https://grafana.com/docs/administration/image_rendering/#troubleshooting).

## Building from source

See [Building from source](https://github.com/grafana/grafana-image-renderer/blob/v1.x/docs/building_from_source.md).

## Additional information

See [docs](https://github.com/grafana/grafana-image-renderer/blob/v1.x/docs/index.md).
