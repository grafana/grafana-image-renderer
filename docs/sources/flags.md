---
description: Flags for the grafana-image-renderer service
keywords:
  - grafana
  - image
  - rendering
  - plugin
  - troubleshooting
labels:
  products:
    - enterprise
    - oss
menuTitle: Flags
title: Image rendering flags
weight: 300
---

# Image rendering flags

This section aims to list the entire set of flags that can be used to configure the image rendering service.

## Configuration format

To configure the service, any of the following methods can be used:

- Set CLI flags. These are passed in on the command line, and take complete precedence over all other methods.
  For example, `--server.addr=":8081"` sets the HTTP address to listen on all interfaces on port `8081`.
- Set environment variables. These are set in the environment.
  If environment variables are supported for a flag, they are listed in the help command.
- Write a JSON or YAML configuration file. These must be named `config.json`, `config.yaml`, or `config.yml`.
  There should only be one file; precedence is undefined if multiple files are present.
  Dot-separated keys are nested keys. For example, the flag `a.b` becomes `{"a": {"b": "VALUE"}}` in the file.
  The configuration keys are always mentioned in the help command alongside the flag.

For example, a complete configuration file might look like this in YAML:

```yaml
server:
  addr: ":8081" # server.addr
  auth-token: # server.auth-token
    - "a"
    - "b"
```

## List of flags

The following is a complete list of all flags that are currently supported in the latest release.
This is a verbatim copy of the output of the `grafana-image-renderer server --help` command.

```
--api.default-encoding=<string> [default: "pdf"] [${API_DEFAULT_ENCODING}]
    The default encoding for render requests when not specified. (values: pdf, png) [config: api.default-encoding]
--browser.flag=<string> / --browser.flags=<string> [${BROWSER_FLAG}]
    Flags to pass to the browser. These are syntaxed `${flag}` or `${flag}=${value}`. [config: browser.flag]
--browser.gpu [default: false] [${BROWSER_GPU}]
    Enable GPU support in the browser. [config: browser.gpu]
--browser.header=<string> / --browser.headers=<string> [${BROWSER_HEADER}]
    Headers to add to every request the browser makes. Syntax is `${key}=${value}`. May be repeated. [config: browser.header]
--browser.max-height=<int> [default: 3000] [${BROWSER_MAX_HEIGHT}]
    The maximum height of the browser viewport. Requests cannot request a larger height than this, except for when capturing full-page screenshots. Negative means ignored. [config: browser.max-height]
--browser.max-width=<int> [default: 3000] [${BROWSER_MAX_WIDTH}]
    The maximum width of the browser viewport. Requests cannot request a larger width than this. Negative means ignored. [config: browser.max-width]
--browser.min-height=<int> [default: 500] [${BROWSER_MIN_HEIGHT}]
    The minimum height of the browser viewport. This is the default height in requests. [config: browser.min-height]
--browser.min-width=<int> [default: 1000] [${BROWSER_MIN_WIDTH}]
    The minimum width of the browser viewport. This is the default width in requests. [config: browser.min-width]
--browser.namespaced [default: false] [${BROWSER_NAMESPACED}]
    Enable namespacing the browser. This requires Linux and the CAP_SYS_ADMIN and CAP_SYS_CHROOT capabilities, or a privileged user. [config: browser.namespaced]
--browser.override=<string> / --browser.overrides=<string> [${BROWSER_OVERRIDE}]
    URL pattern override in format: 'pattern=--flag=value --flag2=value2'. Pattern is a regex. May be repeated. Example: --browser.override='^https://slow\.example\.com/.*=--browser.readiness.timeout=60s' [config: browser.override]
--browser.page-scale-factor=<float> [default: 1] [${BROWSER_PAGE_SCALE_FACTOR}]
    The page scale factor of the browser. [config: browser.page-scale-factor]
--browser.path=<string> [default: "chromium"] [${BROWSER_PATH}]
    The path to the browser's binary. This is resolved against PATH. [config: browser.path]
--browser.portrait [default: false] [${BROWSER_PORTRAIT}]
    Use a portrait viewport instead of the default landscape. [config: browser.portrait]
--browser.readiness.disable-dom-hashcode-wait [default: false] [${BROWSER_READINESS_DISABLE_DOM_HASHCODE_WAIT}]
    Disable waiting for the DOM to stabilize (i.e. not change) before capturing. [config: browser.readiness.disable-dom-hashcode-wait]
--browser.readiness.disable-network-wait [default: false] [${BROWSER_READINESS_DISABLE_NETWORK_WAIT}]
    Disable waiting for network requests to finish before capturing. [config: browser.readiness.disable-network-wait]
--browser.readiness.disable-query-wait [default: false] [${BROWSER_READINESS_DISABLE_QUERY_WAIT}]
    Disable waiting for queries to finish before capturing. [config: browser.readiness.disable-query-wait]
--browser.readiness.dom-hashcode-timeout=<duration> [default: 0s] [${BROWSER_READINESS_DOM_HASHCODE_TIMEOUT}]
    How long to wait before giving up on the DOM stabilizing (i.e. not changing). If <= 0, the timeout is disabled. [config: browser.readiness.dom-hashcode-timeout]
--browser.readiness.give-up-on-all-queries=<duration> [default: 0s] [${BROWSER_READINESS_GIVE_UP_ON_ALL_QUERIES}]
    How long to wait before giving up on all running queries. If <= 0, the give-up is disabled. [config: browser.readiness.give-up-on-all-queries]
--browser.readiness.give-up-on-first-query=<duration> [default: 3s] [${BROWSER_READINESS_GIVE_UP_ON_FIRST_QUERY}]
    How long to wait before giving up on a first query being registered. If <= 0, the give-up is disabled. [config: browser.readiness.give-up-on-first-query]
--browser.readiness.iteration-interval=<duration> [default: 100ms] [${BROWSER_READINESS_ITERATION_INTERVAL}]
    How long to wait between each iteration of checking whether the page is ready. Must be positive. [config: browser.readiness.iteration-interval]
--browser.readiness.network-idle-timeout=<duration> [default: 0s] [${BROWSER_READINESS_NETWORK_IDLE_TIMEOUT}]
    How long to wait before giving up on the network being idle. If <= 0, the timeout is disabled. [config: browser.readiness.network-idle-timeout]
--browser.readiness.prior-wait=<duration> [default: 1s] [${BROWSER_READINESS_PRIOR_WAIT}]
    The time to wait before checking for how ready the page is. This lets you force the webpage to take a beat and just do its thing before the service starts looking for whether it's time to render anything. If <= 0, this is disabled. [config: browser.readiness.prior-wait]
--browser.readiness.timeout=<duration> [default: 30s] [${BROWSER_READINESS_TIMEOUT}]
    The maximum time to wait for a web-page to become ready (i.e. no longer loading anything). If <= 0, the timeout is disabled. [config: browser.readiness.timeout]
--browser.readiness.wait-for-n-query-cycles=<int> [default: 1] [${BROWSER_READINESS_WAIT_FOR_N_QUERY_CYCLES}]
    The number of readiness checks that must pass consecutively before considering the page ready. [config: browser.readiness.wait-for-n-query-cycles]
--browser.sandbox [default: false] [${BROWSER_SANDBOX}]
    Enable the browser's sandbox. Sets the `no-sandbox` flag to `false` for you. [config: browser.sandbox]
--browser.time-between-scrolls=<duration> [default: 50ms] [${BROWSER_TIME_BETWEEN_SCROLLS}]
    The time between scroll events when capturing a full-page screenshot. [config: browser.time-between-scrolls]
--browser.time-zone=<string> / --browser.timezone=<string> / --browser.tz=<string> [default: "Etc/UTC"] [${BROWSER_TIMEZONE}, ${TZ}]
    The timezone for the browser to use, e.g. 'America/New_York'. [config: browser.timezone]
--browser.ws-url-read-timeout=<duration> [default: 0s] [${BROWSER_WS_URL_READ_TIMEOUT}]
    The timeout for reading the WebSocket URL when connecting to the browser. If <= 0, uses chromedp default (20s). [config: browser.ws-url-read-timeout]
--help / -h
    show help
--log.level=<string> [default: "info"] [${LOG_LEVEL}]
    The minimum level to log at (enum: debug, info, warn, error) [config: log.level]
--rate-limit.disabled [default: false] [${RATE_LIMIT_DISABLED}]
    Disable rate limiting entirely. [config: rate-limit.disabled]
--rate-limit.headroom=<uint> [default: 33554432] [${RATE_LIMIT_HEADROOM}]
    The amount of memory (in bytes) to leave as headroom after allocating memory for browser processes. Set to 0 to disable headroom. [config: rate-limit.headroom]
--rate-limit.max-available=<uint> [default: 0] [${RATE_LIMIT_MAX_AVAILABLE}]
    The maximum amount of memory (in bytes) available to processes. If more memory exists, only this amount is used. 0 disables the maximum. [config: rate-limit.max-available]
--rate-limit.max-limit=<uint> [default: 0] [${RATE_LIMIT_MAX_LIMIT}]
    The maximum number of requests to permit. Ratelimiting will reject requests if the number of currently running requests is at or above this value. Set to 0 to disable maximum. The v4 service used 5 by default. [config: rate-limit.max-limit]
--rate-limit.min-limit=<uint> [default: 3] [${RATE_LIMIT_MIN_LIMIT}]
    The minimum number of requests to permit. Ratelimiting will not reject requests if the number of currently running requests is below this value. Set to 0 to disable minimum (not recommended). [config: rate-limit.min-limit]
--rate-limit.min-memory-per-browser=<uint> [default: 67108864] [${RATE_LIMIT_MIN_MEMORY_PER_BROWSER}]
    The minimum amount of memory (in bytes) each browser process is expected to use. Set to 0 to disable the minimum. [config: rate-limit.min-memory-per-browser]
--rate-limit.process-tracker.decay=<int> [default: 5] [${RATE_LIMIT_PROCESS_TRACKER_DECAY}]
    The decay factor N to use in slow-moving averages of process statistics, where `avg = ((N-1)*avg + new) / N`. Must be at least 1. [config: rate-limit.process-tracker.decay]
--rate-limit.process-tracker.interval=<duration> [default: 50ms] [${RATE_LIMIT_PROCESS_TRACKER_INTERVAL}]
    How often to sample process statistics on the browser processes. Must be >= 1ms. [config: rate-limit.process-tracker.interval]
--server.addr=<string> [default: ":8081"] [${SERVER_ADDR}]
    The address to listen on for HTTP requests. [config: server.addr]
--server.auth-token=<string> / --server.auth-tokens=<string> / --server.token=<string> / --server.tokens=<string> [default: "-"] [${AUTH_TOKEN}]
    The X-Auth-Token header value that must be sent to the service to permit requests. May be repeated. [config: server.auth-token]
--server.cert-file=<string> / --server.cert=<string> / --server.certificate-file=<string> / --server.certificate=<string> [${SERVER_CERTIFICATE_FILE}]
    A path to a TLS certificate file to use for HTTPS. If not set, HTTP is used. [config: server.certificate-file]
--server.key-file=<string> / --server.key=<string> [${SERVER_KEY_FILE}]
    A path to a TLS key file to use for HTTPS. [config: server.key-file]
--server.min-tls-version=<string> [default: "1.2"] [${SERVER_MIN_TLS_VERSION}]
    The minimum TLS version to accept for HTTPS connections. (enum: 1.0, 1.1, 1.2, 1.3) [config: server.min-tls-version]
--tracing.client-certificate=<string> [${TRACING_CLIENT_CERTIFICATE}]
    A path to a PEM-encoded client certificate to use for mTLS when connecting to the tracing endpoint over gRPC or HTTPS. [config: tracing.client_certificate]
--tracing.client-key=<string> [${TRACING_CLIENT_KEY}]
    A path to a PEM-encoded client key to use for mTLS when connecting to the tracing endpoint over gRPC or HTTPS. [config: tracing.client_key]
--tracing.compressor=<string> [default: "none"] [${TRACING_COMPRESSOR}]
    The compression algorithm to use when sending traces. (enum: none, gzip) [config: tracing.compressor]
--tracing.endpoint=<string> [${TRACING_ENDPOINT}]
    The tracing endpoint to send spans to. Use grpc://, http://, or https:// to specify the protocol (grpc:// is implied). [config: tracing.endpoint]
--tracing.header=<string> / --tracing.headers=<string> [${TRACING_HEADER}]
    A header to add to requests to the tracing endpoint. Syntax is `${key}=${value}`. May be repeated. This is useful for things like authentication. [config: tracing.header]
--tracing.insecure [default: false] [${TRACING_INSECURE}]
    Whether to skip TLS verification when connecting. If set, the scheme in the endpoint is overridden to be insecure. [config: tracing.insecure]
--tracing.service-name=<string> [default: "grafana-image-renderer"] [${TRACING_SERVICE_NAME}]
    The service name to use in traces. [config: tracing.service_name]
--tracing.timeout=<duration> [default: 10s] [${TRACING_TIMEOUT}]
    The timeout for requests to the tracing endpoint. [config: tracing.timeout]
--tracing.trusted-certificate=<string> [${TRACING_TRUSTED_CERTIFICATE}]
    A path to a PEM-encoded certificate to use as a trusted root when connecting to the tracing endpoint over gRPC or HTTPS. [config: tracing.trusted_certificate]
```

## Debug the configuration

{{< admonition type="note" >}}
The command recommended here prints secrets and sensitive information in
plain text. Be careful when sharing the output or storing it in logs.
{{< /admonition >}}

When the service doesn't appear to be working as expected, it can be because the
configuration is invalid. If the flags are invalid, the service won't start, but
the configuration files may contain unknown keys and thus silently ignore them.

To debug that the configuration is valid and what you expect, you can get a full
dump of the Go structures representing the configuration by running the
`print-config` command. For example:

```
docker run --rm -v ./config.json:/home/nonroot/config.json grafana/grafana-image-renderer:latest print-config
```

The command takes the exact same flags and configuration files as the `server`
command does.
