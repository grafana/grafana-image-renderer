A Grafana backend plugin that handles rendering panels and dashboards to PNGs using headless Chrome.

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

### Environment variables

You can override certain settings by using environment variables and making sure that those are available for the Grafana process.

**Ignore HTTPS errors:**

Instruct headless Chrome Whether to ignore HTTPS errors during navigation. Per default HTTPS errors is not ignored.
Due to the security risk it's not recommended to ignore HTTPS errors.

```bash
export GF_RENDERER_PLUGIN_IGNORE_HTTPS_ERRORS=true
```

**Custom Chrome/Chromium path:**

if you already have Chrome or Chromium installed on your system you can configure Grafana Image renderer plugin to use this instead of the pre-packaged version of Chromium.

Please note that this is not recommended since you may encounter problems if the installed version of Chrome/Chromium is not is compatible with the Grafana Image renderer plugin.

```bash
export GF_RENDERER_PLUGIN_CHROME_BIN=/some/custom/chrome/bin
```

## Remote Rendering Using Docker

Instead of installing and running the image renderer as a plugin, you can run it as a remote image rendering service using Docker. Read more about [remote rendering using Docker](https://github.com/grafana/grafana-image-renderer/blob/master/docs/remote_rendering_using_docker.md).

## Troubleshooting

For troubleshooting help, please refer to [Grafana Image Rendering documentation](https://grafana.com/docs/administration/image_rendering/#troubleshooting).

## Building from source

See [Building from source](https://github.com/grafana/grafana-image-renderer/blob/master/docs/building_from_source.md).

## Additional information

See [docs](https://github.com/grafana/grafana-image-renderer/blob/master/docs/index.md).
