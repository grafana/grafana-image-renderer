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

### Configuration

#### Rendering mode

You can instruct how headless browser instances are created by configuring a rendering mode (`RENDERING_MODE`). Default is `default`, other supported values are `clustered` and `reusable`.

##### Default

Default mode will create a new browser instance on each request. When handling multiple concurrent requests, this mode increases memory usage as it will launch multiple browsers at the same time. If you want to set a maximum

> Note: When using the `default` mode, it's recommended to not remove the default Chromium flag `--disable-gpu`. When receiving a lot of concurrent requests, not using this flag can cause Puppeteer `newPage` function to freeze, causing request timeout and leaving browsers open.

```bash
RENDERING_MODE=default
```

When using `clustered` you can configure a clustering mode to define how many browser instances or incognito pages that can execute concurrently. Default is `browser` and will ensure a maximum amount of browser instances can execute concurrently. Mode `context` will ensure a maximum amount of incognito pages can execute concurrently. You can also configure the maximum concurrency allowed which per default is `5`.

```bash
RENDERING_MODE=clustered
RENDERING_CLUSTERING_MODE=default
RENDERING_CLUSTERING_MAX_CONCURRENCY=5
```

When using the rendering mode `reusable` one browser instance will be created and reused. A new incognito page will be opened on each request for. This mode is a bit experimental since if the browser instance crashes it will not automatically be restarted.

```bash
RENDERING_MODE=reusable
```

#### Configure Grafana

For available Grafana configuration settings, please refer to [Grafana Image Rendering documentation](https://grafana.com/docs/administration/image_rendering/).

## Remote Rendering Using Docker

Instead of installing and running the image renderer as a plugin, you can run it as a remote image rendering service using Docker. Read more about [remote rendering using Docker](https://github.com/grafana/grafana-image-renderer/blob/master/docs/remote_rendering_using_docker.md).

## Troubleshooting

### Performance troubleshooting

### Other

For troubleshooting help, please refer to
[Grafana Image Rendering documentation](https://grafana.com/docs/grafana/latest/administration/image_rendering/#troubleshoot-image-rendering).

## Additional information

See [docs](https://github.com/grafana/grafana-image-renderer/blob/master/docs/index.md).
