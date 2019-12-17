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

**Enable Prometheus metrics:**

You can enable [Prometheus](https://prometheus.io/) metrics endpoint `/metrics` using the environment variable `ENABLE_METRICS`. Node.js and render request duration metrics are included, see [output example](#prometheus-metrics-endpoint-output-example) for details.

```bash
export ENABLE_METRICS=true
```

## Docker Compose example

The following docker-compose example can also be found in [docker/](https://github.com/grafana/grafana-image-renderer/tree/master/devenv/docker/simple).

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

## Prometheus metrics endpoint output example

```
# HELP process_cpu_user_seconds_total Total user CPU time spent in seconds.
# TYPE process_cpu_user_seconds_total counter
process_cpu_user_seconds_total 0.6240000000000001 1576543423268

# HELP process_cpu_system_seconds_total Total system CPU time spent in seconds.
# TYPE process_cpu_system_seconds_total counter
process_cpu_system_seconds_total 0.07200000000000001 1576543423268

# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.6960000000000002 1576543423268

# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1576543243

# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 47038464 1576543423269

# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 1344507904 1576543423269

# HELP process_heap_bytes Process heap size in bytes.
# TYPE process_heap_bytes gauge
process_heap_bytes 1302315008 1576543423269

# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 22 1576543423269

# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 1573877

# HELP nodejs_eventloop_lag_seconds Lag of event loop in seconds.
# TYPE nodejs_eventloop_lag_seconds gauge
nodejs_eventloop_lag_seconds 0.000986571 1576543423269

# HELP nodejs_active_handles Number of active libuv handles grouped by handle type. Every handle type is C++ class name.
# TYPE nodejs_active_handles gauge
nodejs_active_handles{type="Socket"} 4 1576543423268
nodejs_active_handles{type="Server"} 1 1576543423268

# HELP nodejs_active_handles_total Total number of active handles.
# TYPE nodejs_active_handles_total gauge
nodejs_active_handles_total 5 1576543423268

# HELP nodejs_active_requests Number of active libuv requests grouped by request type. Every request type is C++ class name.
# TYPE nodejs_active_requests gauge
nodejs_active_requests{type="FSReqCallback"} 2

# HELP nodejs_active_requests_total Total number of active requests.
# TYPE nodejs_active_requests_total gauge
nodejs_active_requests_total 2 1576543423268

# HELP nodejs_heap_size_total_bytes Process heap size from node.js in bytes.
# TYPE nodejs_heap_size_total_bytes gauge
nodejs_heap_size_total_bytes 12357632 1576543423268

# HELP nodejs_heap_size_used_bytes Process heap size used from node.js in bytes.
# TYPE nodejs_heap_size_used_bytes gauge
nodejs_heap_size_used_bytes 11332920 1576543423268

# HELP nodejs_external_memory_bytes Nodejs external memory size in bytes.
# TYPE nodejs_external_memory_bytes gauge
nodejs_external_memory_bytes 2392542 1576543423268

# HELP nodejs_heap_space_size_total_bytes Process heap space size total from node.js in bytes.
# TYPE nodejs_heap_space_size_total_bytes gauge
nodejs_heap_space_size_total_bytes{space="read_only"} 262144 1576543423268
nodejs_heap_space_size_total_bytes{space="new"} 1048576 1576543423268
nodejs_heap_space_size_total_bytes{space="old"} 8761344 1576543423268
nodejs_heap_space_size_total_bytes{space="code"} 425984 1576543423268
nodejs_heap_space_size_total_bytes{space="map"} 1052672 1576543423268
nodejs_heap_space_size_total_bytes{space="large_object"} 266240 1576543423268
nodejs_heap_space_size_total_bytes{space="code_large_object"} 49152 1576543423268
nodejs_heap_space_size_total_bytes{space="new_large_object"} 491520 1576543423268

# HELP nodejs_heap_space_size_used_bytes Process heap space size used from node.js in bytes.
# TYPE nodejs_heap_space_size_used_bytes gauge
nodejs_heap_space_size_used_bytes{space="read_only"} 32296 1576543423268
nodejs_heap_space_size_used_bytes{space="new"} 1026440 1576543423268
nodejs_heap_space_size_used_bytes{space="old"} 8589920 1576543423268
nodejs_heap_space_size_used_bytes{space="code"} 263680 1576543423268
nodejs_heap_space_size_used_bytes{space="map"} 670640 1576543423268
nodejs_heap_space_size_used_bytes{space="large_object"} 262184 1576543423268
nodejs_heap_space_size_used_bytes{space="code_large_object"} 3552 1576543423268
nodejs_heap_space_size_used_bytes{space="new_large_object"} 486304 1576543423268

# HELP nodejs_heap_space_size_available_bytes Process heap space size available from node.js in bytes.
# TYPE nodejs_heap_space_size_available_bytes gauge
nodejs_heap_space_size_available_bytes{space="read_only"} 229576 1576543423268
nodejs_heap_space_size_available_bytes{space="new"} 21048 1576543423268
nodejs_heap_space_size_available_bytes{space="old"} 148904 1576543423268
nodejs_heap_space_size_available_bytes{space="code"} 25760 1576543423268
nodejs_heap_space_size_available_bytes{space="map"} 380272 1576543423268
nodejs_heap_space_size_available_bytes{space="large_object"} 0 1576543423268
nodejs_heap_space_size_available_bytes{space="code_large_object"} 0 1576543423268
nodejs_heap_space_size_available_bytes{space="new_large_object"} 561184 1576543423268

# HELP nodejs_version_info Node.js version info.
# TYPE nodejs_version_info gauge
nodejs_version_info{version="v12.13.0",major="12",minor="13",patch="0"} 1

# HELP http_request_duration_seconds duration histogram of http responses labeled with: status_code
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.5",status_code="200"} 0
http_request_duration_seconds_bucket{le="1",status_code="200"} 0
http_request_duration_seconds_bucket{le="3",status_code="200"} 3
http_request_duration_seconds_bucket{le="5",status_code="200"} 3
http_request_duration_seconds_bucket{le="7",status_code="200"} 3
http_request_duration_seconds_bucket{le="10",status_code="200"} 3
http_request_duration_seconds_bucket{le="20",status_code="200"} 3
http_request_duration_seconds_bucket{le="30",status_code="200"} 3
http_request_duration_seconds_bucket{le="60",status_code="200"} 3
http_request_duration_seconds_bucket{le="+Inf",status_code="200"} 3
http_request_duration_seconds_sum{status_code="200"} 7.245609712
http_request_duration_seconds_count{status_code="200"} 3

# HELP up 1 = up, 0 = not up
# TYPE up gauge
up 1
```