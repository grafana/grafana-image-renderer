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

## Configuration

### Rendering mode

You can instruct how headless browser instances are created by configuring a rendering mode (`RENDERING_MODE`). Default is `default`, other supported values are `clustered` and `reusable`.

#### Default

Default mode will create a new browser instance on each request. When handling multiple concurrent requests, this mode increases memory usage as it will launch multiple browsers at the same time. If you want to set a maximum number of browser to open, you'll need to use the [clustered mode](#clustered).

> Note: When using the `default` mode, it's recommended to not remove the default Chromium flag `--disable-gpu`. When receiving a lot of concurrent requests, not using this flag can cause Puppeteer `newPage` function to freeze, causing request timeouts and leaving browsers open.

```bash
RENDERING_MODE=default
```

#### Clustered

With the `clustered` mode, you can configure how many browser instances or incognito pages can execute concurrently. Default is `browser` and will ensure a maximum amount of browser instances can execute concurrently. Mode `context` will ensure a maximum amount of incognito pages can execute concurrently. You can also configure the maximum concurrency allowed which per default is `5`. 

Using a cluster of incognito pages is more performant and consumes less CPU and memory than a cluster of browsers. However, if one page crashes it can bring down the entire browser with it (making all the rendering requests happening at the same time fail). Also, each page isn't guaranteed to be totally clean (cookies and storage might bleed-through as seen [here](https://bugs.chromium.org/p/chromium/issues/detail?id=754576)).

```bash
RENDERING_MODE=clustered
RENDERING_CLUSTERING_MODE=default
RENDERING_CLUSTERING_MAX_CONCURRENCY=5
```

#### Reusable (experimental)

When using the rendering mode `reusable`, one browser instance will be created and reused. A new incognito page will be opened for each request. This mode is experimental since, if the browser instance crashes, it will not automatically be restarted. You can achieve a similar behavior using `clustered` mode with a high `maxConcurrency` setting.

```bash
RENDERING_MODE=reusable
```

#### Rendering modes performance and CPU / memory usage

The performance and resources consumption of the different modes depend a lot on the number of concurrent requests your service is handling so it's recommended to monitor your image renderer service. On the Grafana website, you can find a [pre-made dashboard](https://grafana.com/grafana/dashboards/12203) along with documentation on how to set it up. 

With no concurrent request, the different modes show very similar performance and CPU / memory usage. 

When handling concurrent requests, we see the following trends:
- Parallelizing incognito pages instead of browsers is better for performance and CPU / memory consumption meaning that `reusable` and `clustered` (with clustering mode set as `context`) modes are the most performant.
- If you use the `clustered` mode with a `maxConcurrency` setting below your average number of concurrent requests, performance will drop as the rendering requests will need to wait for the other to finish before getting access to an incognito page / browser.

To achieve better performance, it's also recommended to monitor the machine on which your service is running. If you don't have enough memory and / or CPU, every rendering step will be slower than usual, increasing the duration of every rendering request.
Minimum free memory recommendation is 16GB on the system doing the rendering. One advantage of using the remote rendering service is that the rendering will be done on the remote system, so your local system resources will not be affected by rendering. 


### Configure Grafana

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
