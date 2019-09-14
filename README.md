# Grafana Image Renderer [![CircleCI](https://circleci.com/gh/grafana/grafana-image-renderer.svg?style=svg)](https://circleci.com/gh/grafana/grafana-image-renderer)

A Grafana backend plugin that handles rendering panels and dashboards to PNGs using headless Chrome.

## Requirements

### Supported operating systems

- Linux (x64)
- Windows (x64)
- Mac OS X (x64)

### No dependencies

This plugin is packaged in a single executable with [Node.js](https://nodejs.org/) runtime and [Chromium](https://www.chromium.org/Home). It does not require any additional software to be installed on the Grafana server.

## Installation

### Using grafana-cli

**NOTE:** Installing this plugin using grafana-cli is supported from Grafana v6.4.

```
grafana-cli plugins install grafana-image-renderer
```

### Clone into plugins folder

1. Git clone this repo into the Grafana external plugins folder.
2. Install dependencies and build.
<!--- Wait, what? You said in the No dependencies section that there were no dependencies. --->

    ```
    yarn install --pure-lockfile
    yarn run build
    ```

3. Restart Grafana.

## Remote Rendering Using Docker

Instead of installing and running the image renderer as a plugin, you can run it as a remote image rendering service using Docker. Read more about [remote rendering using Docker](https://github.com/grafana/grafana-image-renderer/blob/master/docs/remote_rendering_using_docker.md).

## Troubleshooting

To get more logging information, update the Grafana configuration:

```
[log]
filters = rendering:debug
```

## Additional information

See [docs](https://github.com/grafana/grafana-image-renderer/blob/master/docs/index.md).
