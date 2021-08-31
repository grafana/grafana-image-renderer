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
[Grafana Image Rendering documentation](https://grafana.com/docs/image_rendering/).

### Memory requirements

Minimum free memory recommendation is 16GB on the system doing the rendering.

Rendering images can require a lot of memory, mainly because Grafana creates browser instances in the background for the actual rendering. If multiple images are rendered in parallel, then the rendering has a bigger memory footprint. One advantage of using the remote rendering service is that the rendering will be done on the remote system, so your local system resources will not be affected by rendering.

## Plugin installation

### Using grafana-cli

**NOTE:** Installing this plugin using grafana-cli is supported from Grafana v6.4.

```bash
grafana-cli plugins install grafana-image-renderer
```

### Install in Grafana Docker image

This plugin is not compatible with the current Grafana Docker image without installing further system-level dependencies. We recommend setting up another Docker container for rendering and using remote rendering, see [Run in Docker](#run-in-docker) for reference.

If you still want to install the plugin in the Grafana docker image we provide instructions for how to build a custom Grafana image, see [Grafana Docker documentation](https://grafana.com/docs/installation/docker/#custom-image-with-grafana-image-renderer-plugin-pre-installed) for further instructions.

## Remote rendering service installation

> Requires an internet connection.

This plugin can also be run as a remote HTTP rendering service. In this setup, Grafana renders an image by making a HTTP request to the remote rendering service, which in turn renders the image and returns it back in the HTTP response to Grafana.

You can run the remote HTTP rendering service using Docker or as a standalone Node.js application.

### Run in Docker

The docker images are published at [Docker Hub](https://hub.docker.com/r/grafana/grafana-image-renderer).

The following example shows how to run Grafana and the remote HTTP rendering service in two separate Docker containers using Docker Compose.

Create a `docker-compose.yml` with the following content:

```yaml
version: '2'

services:
  grafana:
    image: grafana/grafana:latest
    ports:
      - '3000:3000'
    environment:
      GF_RENDERING_SERVER_URL: http://renderer:8081/render
      GF_RENDERING_CALLBACK_URL: http://grafana:3000/
      GF_LOG_FILTERS: rendering:debug
  renderer:
    image: grafana/grafana-image-renderer:latest
    ports:
      - 8081
```

And then run:

```bash
docker-compose up
```

### Run as standalone Node.js application

The following example describes how to build and run the remote HTTP rendering service as a standalone Node.js application and configure Grafana appropriately.

1. Clone the [Grafana image renderer plugin](https://grafana.com/grafana/plugins/grafana-image-renderer) Git repository.
1. Install dependencies and build:

   ```bash
   yarn install --pure-lockfile
   yarn run build
   ```

1. Run the server:

   ```bash
   node build/app.js server --port=8081
   ```

1. Update Grafana configuration:

   ```
   [rendering]
   server_url = http://localhost:8081/render
   callback_url = http://localhost:3000/
   ```

1. Restart Grafana.

## Configuration

For available configuration settings, please refer to [Grafana Image Rendering documentation](https://grafana.com/docs/image_rendering/#configuration).

## Troubleshooting

For troubleshooting help, please refer to
[Grafana Image Rendering troubleshooting documentation](https://grafana.com/docs/grafana/latest/administration/image_rendering/troubleshooting.md).
