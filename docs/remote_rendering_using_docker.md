# Remote Rendering Using Docker

As an alternative to installing and running the image renderer as a plugin you can run it as a remote image rendering service using Docker.

The docker image are published at [Docker Hub](https://hub.docker.com/r/grafana/grafana-image-renderer).

## Environment variables

You can override certain settings by using environment variables.

**HTTP host:**

Change the listening host of the HTTP server. Default is unset and will use the local host.

```bash
HTTP_HOST=localhost
```

**HTTP port:**

Change the listening port of the HTTP server. Default is `8081`. Setting `0` will automatically assign a port not in use.

```bash
HTTP_PORT=0
```

**Default timezone:**

Instruct headless browser instance to use a default timezone when not provided by Grafana, .e.g. when rendering panel image of alert. See [ICU’s metaZones.txt](https://cs.chromium.org/chromium/src/third_party/icu/source/data/misc/metaZones.txt?rcl=faee8bc70570192d82d2978a71e2a615788597d1) for a list of supported timezone IDs. Fallbacks to `TZ` environment variable if not set.

```bash
BROWSER_TZ=Europe/Stockholm
```

**Ignore HTTPS errors:**

Instruct headless browser instance whether to ignore HTTPS errors during navigation. Per default HTTPS errors is not ignored.
Due to the security risk it's not recommended to ignore HTTPS errors.

```bash
IGNORE_HTTPS_ERRORS=true
```

**Enable Prometheus metrics:**

You can enable [Prometheus](https://prometheus.io/) metrics endpoint `/metrics` using the environment variable `ENABLE_METRICS`. Node.js and render request duration metrics are included, see [output example](#prometheus-metrics-endpoint-output-example) for details.

```bash
ENABLE_METRICS=true
```

**Log level:**

Change the log level. Default is `info` and will include log messages with level `error`, `warning` and info.

```bash
LOG_LEVEL=debug
```

**Verbose logging:**

Instruct headless browser instance whether to capture and log verbose information when rendering an image. Default is `false` and will only capture and log error messages. When enabled (`true`) debug messages are captured and logged as well.

Note that you need to change log level to `debug`, see above, for the verbose information to be included in the logs.

```bash
RENDERING_VERBOSE_LOGGING=true
```

**Capture browser output:**

Instruct headless browser instance whether to output its debug and error messages into running process of remote rendering service. Default is `false`.
This can be useful to enable (`true`) when troubleshooting.

```bash
RENDERING_DUMPIO=true
```

**Start browser with additional arguments:**

Additional arguments to pass to the headless browser instance. Defaults are `--no-sandbox,--disable-gpu`. The list of Chromium flags can be found [here](https://peter.sh/experiments/chromium-command-line-switches/) and the list of flags used as defaults by Puppeteer can be found [there](https://github.com/puppeteer/puppeteer/blob/main/src/node/Launcher.ts#L172). Multiple arguments is separated with comma-character. 

```bash
RENDERING_ARGS=--no-sandbox,--disable-setuid-sandbox,--disable-dev-shm-usage,--disable-accelerated-2d-canvas,--disable-gpu,--window-size=1280x758
```

**Change how browser instances are created:**

You can instruct how headless browser instances are created by configuring a rendering mode (`RENDERING_MODE`). Default is `default` and will create a new browser instance on each request. Other supported values are `clustered` and `reusable`.

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

## Configuration file

You can override certain settings by using a configuration file, see [default.json](https://github.com/grafana/grafana-image-renderer/tree/master/default.json) for defaults. Note that any configured environment variable takes precedence over configuration file settings.

You can volume mount your custom configuration file when starting the docker container:

```bash
docker run -d --name=renderer --network=host -v /some/path/config.json:/usr/share/grafana/config.json grafana/grafana-image-renderer:latest
```

You can see a docker-compose example using a custom configuration file [here/](https://github.com/grafana/grafana-image-renderer/tree/master/devenv/docker/custom-config).

## Docker Compose example

The following docker-compose example can also be found in [docker/](https://github.com/grafana/grafana-image-renderer/tree/master/devenv/docker/simple).

```bash
version: '2'

services:
  grafana:
    image: grafana/grafana:latest
    ports:
      - 3000
    environment:
      GF_RENDERING_SERVER_URL: http://renderer:8081/render
      GF_RENDERING_CALLBACK_URL: http://grafana:3000/
      GF_LOG_FILTERS: rendering:debug
  renderer:
    image: grafana/grafana-image-renderer:latest
    ports:
      - 8081
    environment:
      ENABLE_METRICS: 'true'
```

1. Start containers:
    ```bash
    docker-compose up
    ```
2. Open http://localhost:3000
3. Create a dashboard, add a panel and save the dashboard.
4. Panel context menu -> Share -> Direct link rendered image

## Enable Prometheus metrics endpoint

The service can be configured to expose a Prometheus metrics endpoint. There's [dashboard](https://grafana.com/grafana/dashboards/12203) published that explains the details of how to configure and monitor the rendering service using Prometheus as a data source.

**Metrics endpoint output example:**

```
# HELP process_cpu_user_seconds_total Total user CPU time spent in seconds.
# TYPE process_cpu_user_seconds_total counter
process_cpu_user_seconds_total 0.536 1579444523566

# HELP process_cpu_system_seconds_total Total system CPU time spent in seconds.
# TYPE process_cpu_system_seconds_total counter
process_cpu_system_seconds_total 0.064 1579444523566

# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.6000000000000001 1579444523566

# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1579444433

# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 52686848 1579444523568

# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 2055344128 1579444523568

# HELP process_heap_bytes Process heap size in bytes.
# TYPE process_heap_bytes gauge
process_heap_bytes 1996390400 1579444523568

# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 31 1579444523567

# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1573877

# HELP nodejs_eventloop_lag_seconds Lag of event loop in seconds.
# TYPE nodejs_eventloop_lag_seconds gauge
nodejs_eventloop_lag_seconds 0.000915922 1579444523567

# HELP nodejs_active_handles Number of active libuv handles grouped by handle type. Every handle type is C++ class name.
# TYPE nodejs_active_handles gauge
nodejs_active_handles{type="WriteStream"} 2 1579444523566
nodejs_active_handles{type="Server"} 1 1579444523566
nodejs_active_handles{type="Socket"} 9 1579444523566
nodejs_active_handles{type="ChildProcess"} 2 1579444523566

# HELP nodejs_active_handles_total Total number of active handles.
# TYPE nodejs_active_handles_total gauge
nodejs_active_handles_total 14 1579444523567

# HELP nodejs_active_requests Number of active libuv requests grouped by request type. Every request type is C++ class name.
# TYPE nodejs_active_requests gauge
nodejs_active_requests{type="FSReqCallback"} 2

# HELP nodejs_active_requests_total Total number of active requests.
# TYPE nodejs_active_requests_total gauge
nodejs_active_requests_total 2 1579444523567

# HELP nodejs_heap_size_total_bytes Process heap size from node.js in bytes.
# TYPE nodejs_heap_size_total_bytes gauge
nodejs_heap_size_total_bytes 13725696 1579444523567

# HELP nodejs_heap_size_used_bytes Process heap size used from node.js in bytes.
# TYPE nodejs_heap_size_used_bytes gauge
nodejs_heap_size_used_bytes 12068008 1579444523567

# HELP nodejs_external_memory_bytes Nodejs external memory size in bytes.
# TYPE nodejs_external_memory_bytes gauge
nodejs_external_memory_bytes 1728962 1579444523567

# HELP nodejs_heap_space_size_total_bytes Process heap space size total from node.js in bytes.
# TYPE nodejs_heap_space_size_total_bytes gauge
nodejs_heap_space_size_total_bytes{space="read_only"} 262144 1579444523567
nodejs_heap_space_size_total_bytes{space="new"} 1048576 1579444523567
nodejs_heap_space_size_total_bytes{space="old"} 9809920 1579444523567
nodejs_heap_space_size_total_bytes{space="code"} 425984 1579444523567
nodejs_heap_space_size_total_bytes{space="map"} 1052672 1579444523567
nodejs_heap_space_size_total_bytes{space="large_object"} 1077248 1579444523567
nodejs_heap_space_size_total_bytes{space="code_large_object"} 49152 1579444523567
nodejs_heap_space_size_total_bytes{space="new_large_object"} 0 1579444523567

# HELP nodejs_heap_space_size_used_bytes Process heap space size used from node.js in bytes.
# TYPE nodejs_heap_space_size_used_bytes gauge
nodejs_heap_space_size_used_bytes{space="read_only"} 32296 1579444523567
nodejs_heap_space_size_used_bytes{space="new"} 601696 1579444523567
nodejs_heap_space_size_used_bytes{space="old"} 9376600 1579444523567
nodejs_heap_space_size_used_bytes{space="code"} 286688 1579444523567
nodejs_heap_space_size_used_bytes{space="map"} 704320 1579444523567
nodejs_heap_space_size_used_bytes{space="large_object"} 1064872 1579444523567
nodejs_heap_space_size_used_bytes{space="code_large_object"} 3552 1579444523567
nodejs_heap_space_size_used_bytes{space="new_large_object"} 0 1579444523567

# HELP nodejs_heap_space_size_available_bytes Process heap space size available from node.js in bytes.
# TYPE nodejs_heap_space_size_available_bytes gauge
nodejs_heap_space_size_available_bytes{space="read_only"} 229576 1579444523567
nodejs_heap_space_size_available_bytes{space="new"} 445792 1579444523567
nodejs_heap_space_size_available_bytes{space="old"} 417712 1579444523567
nodejs_heap_space_size_available_bytes{space="code"} 20576 1579444523567
nodejs_heap_space_size_available_bytes{space="map"} 343632 1579444523567
nodejs_heap_space_size_available_bytes{space="large_object"} 0 1579444523567
nodejs_heap_space_size_available_bytes{space="code_large_object"} 0 1579444523567
nodejs_heap_space_size_available_bytes{space="new_large_object"} 1047488 1579444523567

# HELP nodejs_version_info Node.js version info.
# TYPE nodejs_version_info gauge
nodejs_version_info{version="v14.16.1",major="14",minor="16",patch="1"} 1

# HELP grafana_image_renderer_service_http_request_duration_seconds duration histogram of http responses labeled with: status_code
# TYPE grafana_image_renderer_service_http_request_duration_seconds histogram
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="1",status_code="200"} 0
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="5",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="7",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="9",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="11",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="13",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="15",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="20",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="30",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_bucket{le="+Inf",status_code="200"} 4
grafana_image_renderer_service_http_request_duration_seconds_sum{status_code="200"} 10.492873834
grafana_image_renderer_service_http_request_duration_seconds_count{status_code="200"} 4

# HELP up 1 = up, 0 = not up
# TYPE up gauge
up 1

# HELP grafana_image_renderer_browser_info A metric with a constant '1 value labeled by version of the browser in use
# TYPE grafana_image_renderer_browser_info gauge
grafana_image_renderer_browser_info{version="HeadlessChrome/79.0.3945.0"} 1
```