# Remote Rendering Using Docker

As an alternative to installing and running the image renderer as a plugin you can run it as a remote image rendering service using Docker.

The docker image are published at [Docker Hub](https://hub.docker.com/r/grafana/grafana-image-renderer).

## Environment variables

You can override certain settings by using environment variables and making sure that those are available for the Grafana process.

**Ignore HTTPS errors:**

Instruct headless Chrome Whether to ignore HTTPS errors during navigation. Per default HTTPS errors is not ignored.
Due to the security risk it's not recommended to ignore HTTPS errors.

```bash
export IGNORE_HTTPS_ERRORS=true
```

**Custom Chrome/Chromium path:**

if you already have Chrome or Chromium installed on your system you can configure Grafana Image renderer plugin to use this instead of the pre-packaged version of Chromium.

Please note that this is not recommended since you may encounter problems if the installed version of Chrome/Chromium is not is compatible with the Grafana Image renderer plugin.

```bash
export CHROME_BIN=/some/custom/chrome/bin
```

## Docker Compose example

The following docker-compose example can also be found in [docker/](https://github.com/grafana/grafana-image-renderer/tree/master/docker).

```bash
version: '2'

services:
  grafana:
    image: grafana/grafana:master
    ports:
     - "3000:3000"
    environment:
      GF_RENDERING_SERVER_URL: http://renderer:8081/render
      GF_RENDERING_CALLBACK_URL: http://grafana:3000/
      GF_LOG_FILTERS: rendering:debug
  renderer:
    image: grafana/grafana-image-renderer:latest
    ports:
      - 8081
```

1. Start containers:
    ```bash
    docker-compose up
    ```
2. Open http://localhost:3000
3. Create a dashboard, add a panel and save the dashboard.
4. Panel context menu -> Share -> Direct link rendered image
