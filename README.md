A Grafana backend plugin that handles rendering panels and dashboards to PNGs using headless Chrome.

## Requirements

### Supported operating systems

- Linux (x64)
- Windows (x64)
- Mac OS X (x64)

### Dependencies

This plugin is packaged in a single executable with [Node.js](https://nodejs.org/) runtime and [Chromium browser](https://www.chromium.org/Home).
This means that you don't have to have Node.js and Chromium installed in your system for the plugin to function.

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

## Remote Rendering Using Docker

Instead of installing and running the image renderer as a plugin, you can run it as a remote image rendering service using Docker. Read more about [remote rendering using Docker](https://github.com/grafana/grafana-image-renderer/blob/master/docs/remote_rendering_using_docker.md).

## Troubleshooting

For troubleshooting help, please refer to [Grafana Image Rendering documentation](https://grafana.com/docs/administration/image_rendering/#troubleshooting).

## Building from source

Git clone this repo:

```bash
git clone https://github.com/grafana/grafana-image-renderer.git
cd grafana-image-renderer
```

### Plugin

1. Install [Node.js](https://nodejs.org/) and [Yarn](https://yarnpkg.com/en/).
2. Install dependencies and build:

    ```bash
    make deps

    # build and package for Linux x64
    make build_package ARCH=linux-x64-glibc

    # build and package for Window x64
    make build_package ARCH=win32-x64-unknown

    # build and package for Darwin x64
    make build_package ARCH=darwin-x64-unknown

    # build and package without including Chromium
    make build_package ARCH=<ARCH> SKIP_CHROMIUM=true OUT=plugin-<ARCH>-no-chromium
    ```

3. Built artifacts can be found in ./artifacts directory

### Docker image

1. Install Docker
2. Build Docker image:

    ```bash
    docker build -t custom-grafana-image-renderer .
    ```

### Local Node.js application using local Chrome/Chromium

1. Install [Node.js](https://nodejs.org/) and [Yarn](https://yarnpkg.com/en/).
2. Install dependencies and build:

    ```bash
    make deps
    make build
    ```

3. Built artifacts are found in ./build

## Additional information

See [docs](https://github.com/grafana/grafana-image-renderer/blob/master/docs/index.md).
