# Troubleshooting

## What options do I even have?

See `docker run --rm grafana/grafana-image-renderer:latest server --help`.
Many options you see listed in this repository will have additional options you can use to tweak the exact behaviour more to your needs.

## How do I use the config file?

Write a JSON or YAML configuration file named one of `config.json`, `config.yaml`, or `config.yml` in the current working directory of the service.
The default directory is `/home/nonroot/`.
You can write YAML in the `.json` file: all JSON is valid YAML, so we parse all as YAML.
Kubernetes' own YAML format (KYAML) is valid.

## How do I monitor this service?

You can monitor the service via Prometheus/[Mimir](https://grafana.com/oss/mimir) and any OpenTelemetry-compatible Tracing backend (like [Grafana Tempo](https://grafana.com/oss/tempo)).
We recommend you set up both:

- Point your metrics scraper to `/metrics` on the HTTP port (default `:8081`).
- Point the service (`--tracing.endpoint`) to your tracing backend.

## I need to change the address

See `--server.addr`. If no specific address is given, it will listen on all interfaces.
The syntax to only change port is `:8081` (or any other port number).

## I want to use multiple authentication tokens

Specify the option multiple times, e.g.: `--server.auth-token token1 -server.auth-token token2`.

If you use JSON or YAML, you can use a list:

```yaml
server:
  auth-token:
    - token1
    - token2
```

For environment variables, use a comma-separated list.

## I want to change the logging level

You can set the level with `--log.level`. Valid values are `debug`, `info`, `warn`, and `error`.
Please be prepared for `debug` being _very_ verbose.
Production deployments should usually use `info` or `warn`.

## I need to use TLS

You can set this up with `--server.certificate-file`, `--server.key-file`, and optionally, `--server.min-tls-version`.
mTLS is not supported at this time.

## My tracing needs mTLS

You can set this up with `--tracing.trusted-certificate`, `--tracing.client-certificate`, and `--tracing.client-key`.

## I need to change the browser path

You can use any browser binary with `--browser.path`.
The browser must support the Chrome DevTools Protocol, limiting your choices somewhat.
This works fine with Chromium, Google Chrome, Microsoft Edge, Brave, and other similar browsers based on Chromium.
We only officially support Chromium; if you cannot replicate your bug with it, it may be deprioritised or closed without a fix.

## I have a GPU I want to use

Pass `--browser.gpu`. This may need additional configuration depending on your environment.

## I want custom flags for my browser

Pass `--browser.flag`.
You can pass this multiple times.
The `--` prefix of flags is optional.
If your flag has a value, use `${flag}=${value}`.

## I want to enable the Chrome sandbox

Pass in `--browser.sandbox`.
This is not supported in all environments.

## I want to use Linux namespaces for better isolation

Pass in `--browser.namespaced`.
This is unsupported; if you want to report a bug, please disable this first.
This requires Linux.

## I want to change the default timezone

Pass in `--browser.timezone` with an IANA timezone name, e.g. `America/Los_Angeles` or `Europe/Berlin`.
Note that requests can also set this.

## I want to add a header to all requests

Pass in `--browser.header`.
Note that this may break with CORS.

## I want to pass my trace through to the browser

This is done by default.
All requests also get a `Traceparent` header.

## I get an incomplete page, or all requests time out

The browser waits for the web page to become ready.
This is done by waiting for all of the following to complete or time out:

- Scrolls all the webports (i.e. to load the entire page).
  - Every scroll waits `--browser.scroll-wait` (default 50ms) afterwards.
- Wait `--browser.readiness.prior-wait` (default 1s).
- The entire following sequence times out after `--browser.readiness.timeout` (default 30s):
  - Repeat the checks every `--browser.readiness.interval` (default 100ms).
  - Wait for all Grafana queries to complete, unless `--browser.readiness.disable-query-wait`. This requires Scenes to be enabled.
    Times out after `--browser.readiness.give-up-on-all-queries` (default 0s, meaning disabled).
    The first query must happen within `--browser.readiness.give-up-on-first-query` (default 3s), otherwise we ignore the query check.
  - Wait for all network requests to complete, unless `--browser.readiness.disable-network-wait`.
    Times out after `--browser.readiness.network-idle-timeout` (default 0s, meaning disabled).
  - Wait for the webpage DOM to stabilise, unless `--browser.readiness.disable-dom-hashcode-wait`.
    Times out after `--browser.readiness.dom-hashcode-timeout` (default 0s, meaning disabled).

## Go eats up all the memory in my container

Set `GOMEMLIMIT` to a lower value than your container limit, such as `1GiB`.
You should not aim for the `GOMEMLIMIT` to match the container's limit: Chromium needs free memory on top.
We recommend 1 GiB of `GOMEMLIMIT` per 8 GiB of container memory limit.

## You do not support an architecture I use

Sorry about that. Open an issue, or compile it yourself.
See the [README.md](./README.md) for instructions.

## I'm airgapped. How do I use this?

You need to import the Docker image via a USB stick or similar.
If you are a Grafana Enterprise customer, consider contacting Grafana Support.

## My Grafana isn't in Docker

You can use [host networking](https://docs.docker.com/engine/network/tutorials/host/) instead.

## I use Windows and do not want Docker

You can download the Windows binaries from the GitHub Release.
As an example, this is how you run it with Brave on an ARM64 Windows host:

```powershell
PS > .\grafana-image-renderer-windows-arm64.exe server --browser.path "C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe"
```

The browser must be installed separately.
The browser must be a Chromium-based browser.
