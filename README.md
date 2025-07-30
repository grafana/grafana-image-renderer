A Grafana backend plugin that handles rendering panels and dashboards to PNGs using a headless browser (Chromium).

## Requirements

### Supported operating systems

- Linux (x64)
- Windows (x64)
- Mac OS X (x64)

For Mac ARM64, you need to [build the plugin from source](https://github.com/grafana/grafana-image-renderer/blob/master/docs/building_from_source.md) or use the [remote rendering installation](https://github.com/grafana/grafana-image-renderer?tab=readme-ov-file#remote-rendering-service-installation).

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

## Installation

We offer two installation methods: as a plugin and as a remote service. We always recommend using the remote service if possible, as this is how we deploy the service in Grafana Cloud, and thus gets the most attention in bug handling.

**I deploy Grafana in Docker/Kubernetes/...:** Use the Docker image.

**I deploy Grafana as a user/in systemd/...:**

  * Prefer the Docker image with `--networking=host` passed to the Docker container.
  * If that is not fitting, prefer the standalone server.
  * If that is not fitting, use the plugin.

### Docker image (recommended)

If you want to run the service as a Docker container, use the Docker image we publish [to DockerHub][image].

With `docker run`:

```shell
$ docker network create grafana
$ docker run --network grafana --name renderer --rm --detach grafana/grafana-image-renderer:latest
# The following is not a production-ready Grafana instance, but shows what env vars you should set:
$ docker run --network grafana --name grafana --rm --detach --env GF_RENDERING_SERVER_URL=http://renderer:8081/render --env http://grafana:3000/ --port 3000:3000 grafana/grafana-enterprise:latest
```

With `docker compose`:

```yaml
services:
  renderer:
    image: grafana/grafana-image-renderer:latest

  grafana:
    image: grafana/grafana-enterprise:latest
    ports:
      - '3000:3000'
    environment:
      GF_RENDERING_SERVER_URL: http://renderer:8081/render
      GF_RENDERING_CALLBACK_URL: http://grafana:3000/
```

With Kubernetes, see our [k3s setup](./devenv/k3s/grafana.yaml).

[image]: https://hub.docker.com/r/grafana/grafana-image-renderer

### Run as standalone Node.js application

The following example describes how to build and run the remote HTTP rendering service as a standalone Node.js application and configure Grafana appropriately.

1. Clone the [Grafana image renderer plugin](https://github.com/grafana/grafana-image-renderer/) Git repository.
2. Install dependencies and build:

   ```bash
   yarn install --pure-lockfile
   yarn run build
   ```

3. Run the server:
   - Using default configuration: `node build/app.js server`
   - Using custom [configuration](https://grafana.com/docs/grafana/latest/image-rendering/#configuration): `node build/app.js server --config=dev.json`
   - Using environment variables: `HTTP_PORT=8085 LOG_LEVEL=debug node build/app.js server`

4. Update Grafana configuration:

   ```
   [rendering]
   server_url = http://localhost:8081/render
   callback_url = http://localhost:3000/
   ```

1. Restart Grafana.

### Plugin: Grafana CLI

You can install the plugin with Grafana CLI:

```shell
$ grafana cli plugins install grafana-image-renderer
# alternatively, if you want to install a specific version:
$ grafana cli plugins install grafana-image-renderer $VERSION
```

Please run this as the same user that Grafana runs as, otherwise the plugin may not work!

## Security

Access to the rendering endpoints is restricted to requests providing an auth token. This token should be configured in the Grafana configuration file and the renderer configuration file. This token is important when you run the plugin in remote rendering mode to avoid unauthorized file disclosure (see [CVE-2022-31176](https://github.com/grafana/grafana-image-renderer/security/advisories/GHSA-2cfh-233g-m4c5)).

See [Grafana Image Rendering documentation](https://grafana.com/docs/grafana/latest/image-rendering/#security) to configure this secret token. The default value `-` is configured on both Grafana and the image renderer when you get started but we strongly recommend you to update this to a more secure value.

## Configuration

For available configuration settings, please refer to [Grafana Image Rendering documentation](https://grafana.com/docs/grafana/latest/image-rendering/#configuration).

## Troubleshooting

For troubleshooting help, refer to
[Grafana Image Rendering troubleshooting documentation](https://grafana.com/docs/grafana/latest/image-rendering/troubleshooting/).
