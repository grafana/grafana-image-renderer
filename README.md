# Grafana Image Renderer [![CircleCI](https://circleci.com/gh/grafana/grafana-image-renderer.svg?style=svg)](https://circleci.com/gh/grafana/grafana-image-renderer)

A Grafana Backend Plugin that handles rendering panels &amp; dashboards to PNGs using headless chrome.

## Requirements

### Supported operating systems

- Linux (x64)
- Windows (x64)
- Mac OS X (x64)

### No dependencies

This plugin have been packaged into a single executable together with [Node.js](https://nodejs.org/) runtime and [Chromium](https://www.chromium.org/Home) so it doesn't require any additional dependencies to be installed on the Grafana server.

## Installation

### Using grafana-cli

NOTE: Installing this plugin using grafana-cli is supported from Grafana v6.4.

```
grafana-cli plugins install grafana-image-renderer
```

### Clone into plugins folder

1. git clone into Grafana external plugins folder.
2. Install dependencies and build

    ```
    yarn install --pure-lockfile
    yarn run build
    ```

3. Restart Grafana

## Remote Rendering Using Docker

As an alternative to installing and running the image renderer as a plugin you can run it as a remote image rendering service using Docker. Read more [here](https://github.com/grafana/grafana-image-renderer/blob/master/docs/remote_rendering_using_docker.md).

## Troubleshooting

To get more logging information, update Grafana configuration:

```
[log]
filters = rendering:debug
```

## Additional information

See [docs](https://github.com/grafana/grafana-image-renderer/blob/master/docs/index.md).