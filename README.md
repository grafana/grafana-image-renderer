A Grafana backend plugin that handles rendering panels and dashboards to PNGs using a headless browser (Chromium).

## Requirements

### Supported operating systems

- Linux (x64)
- Windows (x64)
- Mac OS X (x64)

### Dependencies

This plugin is packaged in a single executable with [Node.js](https://nodejs.org/) runtime and [Chromium browser](https://www.chromium.org/Home).
This means that you don't need to have Node.js and Chromium installed in your system for the plugin to function.

However, the [Chromium browser](https://www.chromium.org/) depends on certain libraries. If you don't have all of those libraries installed in your
system, you may see some errors when you try to render an image. For more information including troubleshooting help, refer to
[Grafana Image Rendering documentation](https://grafana.com/docs/grafana/latest/image-rendering/).

### Memory requirements

Rendering images requires a lot of memory, mainly because Grafana creates browser instances in the background for the actual rendering.
We recommend a minimum of 16GB of free memory on the system rendering images.

Rendering multiple images in parallel requires an even bigger memory footprint. You can use the remote rendering service in order to render images on a remote system, so your local system resources are not affected.

## Plugin installation

You can install the plugin using Grafana CLI (recommended way) or with Grafana Docker image.

### Grafana CLI (recommended)

```bash
grafana-cli plugins install grafana-image-renderer
```

### Grafana Docker image

This plugin is not compatible with the current Grafana Docker image and requires additional system-level dependencies. We recommend setting up another Docker container for rendering and using remote rendering instead. For instruction, refer to [Run in Docker](#run-in-docker).

If you still want to install the plugin with the Grafana Docker image, refer to the instructions on building a custom Grafana image in [Grafana Docker documentation](https://grafana.com/docs/installation/docker/#custom-image-with-grafana-image-renderer-plugin-pre-installed).

## Remote rendering service installation

> **Note:** Requires an internet connection.

You can run this plugin as a remote HTTP rendering service. In this setup, Grafana renders an image by making an HTTP request to the remote rendering service, which in turn renders the image and returns it back in the HTTP response to Grafana.

You can run the remote HTTP rendering service using Docker or as a standalone Node.js application.

### Run in Docker

Grafana Docker images are published at [Docker Hub](https://hub.docker.com/r/grafana/grafana-image-renderer).

The following example shows how you can run Grafana and the remote HTTP rendering service in two separate Docker containers using Docker Compose.

1. Create a `docker-compose.yml` with the following content:

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

1. Next, run docker compose.

    ```bash
    docker-compose up
    ```

### Run as standalone Node.js application

The following example describes how to build and run the remote HTTP rendering service as a standalone Node.js application and configure Grafana appropriately.

1. Clone the [Grafana image renderer plugin](https://github.com/grafana/grafana-image-renderer/) Git repository.
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

## Security

Access to the rendering endpoints is restricted to requests providing an auth token. This token should be configured in the Grafana configuration file and the renderer configuration file. This token is important when you run the plugin in remote rendering mode to avoid unauthorized file disclosure (see [CVE-2022-31176](https://github.com/grafana/grafana-image-renderer/security/advisories/GHSA-2cfh-233g-m4c5)).

See [Grafana Image Rendering documentation](https://grafana.com/docs/grafana/latest/image-rendering/#security) to configure this secret token. The default value `-` is configured on both Grafana and the image renderer when you get started but we strongly recommend you to update this to a more secure value.

## Configuration

For available configuration settings, please refer to [Grafana Image Rendering documentation](https://grafana.com/docs/grafana/latest/image-rendering/#configuration).

## Troubleshooting

For troubleshooting help, refer to
[Grafana Image Rendering troubleshooting documentation](https://grafana.com/docs/grafana/latest/image-rendering/troubleshooting/).
