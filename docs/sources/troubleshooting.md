---
aliases:
  - ../../image-rendering/troubleshooting/
  - monitoring
description: Image rendering troubleshooting
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
menuTitle: Troubleshooting
title: Troubleshoot image rendering
weight: 200
---

# Troubleshooting

This section is dedicated to answering some of our more common questions. These
are helpful for self-managed Grafana users; they are not particularly useful for
Grafana Cloud users, as we manage all this stuff for you.

## Available configuration options

To see all available options, run:

```shell
docker run --rm grafana/grafana-image-renderer:latest server --help
```

Much of the service's functionality has fine-tunable options. When the image
renderer service is likely to be at fault, these options tend to be the first
ones that should be changed.

The rest of this document aims to clarify what these options are and do, so the
right experiments and changes can be done.

## Configuration file formats and paths

The configuration files are read from the current working directory of the
service. With our Docker images, this is `/home/nonroot/` by default.

The file names must be one of `config.json`, `config.yaml`, or `config.yml`.

## Monitor the image renderer

You can monitor the service via Prometheus or
[Mimir](https://grafana.com/oss/mimir) and any OpenTelemetry-compatible Tracing
backend, such as [Grafana Tempo](https://grafana.com/oss/tempo). We recommend
setting up both:

- Point the metrics scraper to `/metrics` on the HTTP port (default `:8081`).
- Point the service (`--tracing.endpoint`) to the tracing backend.

We have an [example dashboard here](https://grafana.com/grafana/dashboards/12203-grafana-image-renderer/).

## Changing the HTTP server bind address

Refer to the `--server.addr`.

If no specific address is given, it listens on all interfaces. The syntax to
only change the port is `:8081` or any other port number.

## Using multiple authentication tokens

Specify the option multiple times, for example: `--server.auth-token <token1>
--server.auth-token <token2>`.

If you use JSON or YAML, you can use a list:

```yaml
server:
  auth-token:
    - <token1>
    - <token2>
```

For environment variables, use a comma-separated list.

## Changing the logging level

The log level is changed with the `--log.level` option.

Valid values are `debug`, `info`, `warn`, and `error`. `debug` is _very_
verbose. Production deployments should usually use `info` or `warn`.

## Setting up TLS on the HTTP server

You can serve the HTTP server over TLS (HTTPS) through the following options:

- `--server.certificate-file`: path to the TLS certificate file in PEM format.
- `--server.key-file`: path to the TLS private key file in PEM format. This must
  be the matching key for the given certificate file.
- `--server.min-tls-version`: minimum TLS version to accept. Valid values are
  `1.0`, `1.1`, `1.2` (default), and `1.3`. The default value is sufficient
  for most security-concious users.

Mutual TLS (mTLS) is not supported at this time.

## Setting up mTLS with the tracing backend

You can set up mTLS for the connection to the tracing backend through the
following options:

- `--tracing.trusted-certificate`: path to the trusted CA certificate file in
  PEM format. This is used to verify the tracing backend's certificate.
- `--tracing.client-certificate`: path to the client certificate file in PEM
  format. This is used to authenticate to the tracing backend.
- `--tracing.client-key`: path to the client private key file in PEM format.
  This must be the matching key for the given client certificate file.

## Using a custom browser binary

The browser binary is set with the `--browser.path` option.

The browser must support the Chrome DevTools Protocol, limiting the choices
somewhat. This works fine with Chromium, Google Chrome, Microsoft Edge, Brave,
and other similar browsers based on Chromium.

Only Chromium is officially supported. If a bug cannot be replicated with
Chromium, the bug may be not be prioritised or closed without a fix.

## Using GPU acceleration

Enable GPU acceleration in the browser through the `--browser.gpu` option.

On some environments, such as in Docker, Kubernetes, or other container and VM
runtimes, further configuration may be required to pass the GPU through to the
service.

## Enabling custom flags in the browser

The flags can be passed through with the `--browser.flag` option. The flag is
repeatable, meaning you can pass multiple flags by specifying the option many
times.

The format is `${flag}=${value}`. The `--` prefix is added if it is not present.
For example, `--browser.flag --headless=false` enables headful mode.

## Enabling the browser sandbox

Enable the sandbox through the `--browser.sandbox` option.

On some Linux distributions, in Docker, Kubernetes, OpenShift, and other
container and VM setups, this may not work out of the box. Here, it may be
required to enable various virtualisation features, `seccomp` profiles, AppArmor
profiles, Linux capabilities, and more.

## Using Linux namespaces for request isolation

{{< admonition type="caution" >}}
Although there is an option to enable Linux namespaces, the functionality is
unsupported. Proceed at your own risk, and ensure the option is disabled before
reporting bugs.
{{< /admonition >}}

To use new Linux namespaces for each rendering request, isolating the entire
browser from the service and other requests, use the `--browser.namespaced`
option.

This functionality requires Linux and various capabilities and AppArmor profiles
to be set up.

## Changing the default browser time zone

{{< admonition type="note" >}}
Every request can override this in the request query parameters.
{{< /admonition >}}

To change the default timezone, set the `--browser.timezone` option to an IANA
time zone name. For example, `America/Los_Angeles` or `Europe/Berlin`.

Many containers automatically set a `TZ` environment variable. This is used by
default.

## Adding a header to every request from the browser

{{< admonition type="note" >}}
Depending on the target website's CORS settings, this may break all requests.
{{< /admonition >}}

You can set a new header with `--browser.header <name>=<value>`. This header is
added to every request made by the browser.

## Passing through a trace header to every request from the browser

This is enabled by default if tracing is set up. Outgoing requests receive a
`Traceparent` header.

If the incoming request to the service has a `Traceparent` header, that value is
used. Otherwise, a new trace is started for every request.

## Understanding incomplete outputs and request timeouts

The browser waits for the web page to become ready. This is done by waiting for
all of the following to complete or time out:

- Scrolls all the web-ports (that is, to load the entire page).
  - After every scroll, the browser waits the duration declared by the
    `--browser.scroll-wait` option, 50 milliseconds by default.
- The browser waits the duration declared by the
  `--browser.readiness.prior-wait` option, 1 second by default.
- The entire following sequence times out after the duration declared by the
  `--browser.readiness.timeout` option, 30 seconds by default:
  - The sequence is repeated per duration declared by the
    `--browser.readiness.interval` option, 100 milliseconds by default.
  - The browser waits for all Grafana queries to complete, unless the
    `--browser.readiness.disable-query-wait` option is enabled. This
    functionality requires Scenes to be enabled. If Scenes are not enabled, the
    check is skipped silently.
    - If the queries do not complete within the duration declared by the
      `--browser.readiness.give-up-on-all-queries` option, the check is silently
      skipped. By default, this timeout is disabled.
    - If there is no first query detected within the duration declared by the
      `--browser.readiness.give-up-on-first-query` option, the check is silently
      skipped. By default, this timeout is 3 seconds.
  - The browser waits for all network requests to complete, unless the
    `--browser.readiness.disable-network-wait` option is enabled.
    - If the network requests do not complete within the duration declared by the
      `--browser.readiness.network-idle-timeout` option, the check is silently
      skipped. By default, this timeout is disabled.
  - The browser waits for the web page's layout to stabilise, meaning no more
    data changes. This can be disabled through the
    `--browser.readiness.disable-dom-hashcode-wait` option.
    - If the web page does not stabilise within the duration declared by the
      `--browser.readiness.dom-hashcode-timeout` option, the check is silently
      skipped. By default, this timeout is disabled.

## The service eats up all the memory in the container

Set the `GOMEMLIMIT` environment variable to a lower value than the container's
memory limit, such as `1GiB`. The value should not be the same as the
container's memory limit, because Chromium needs free memory to serve requests.

We recommend adding 1 GiB to this environment variable for every 8 GiB of memory
assigned to the container's memory limit.

## Unsupported CPU architectures

For unsupported CPU architectures, you can open an issue on [GitHub].

Alternatively, compile the service yourself, following the instructions in the
[GitHub repository][github].

[github]: https://github.com/grafana/grafana-image-renderer

## Using the image renderer service in an air-gapped environment

An air-gapped environment is one that does not have access to the public
internet, and may have requirements such as not supporting Docker.

Grafana Enterprise customers can receive more help with this from customer
support.

### With Docker

You will need:

- A way to transfer data into the environment, such as a USB stick, SD card,
  external hard-drive, or similar.
- Docker on the air-gapped environment.
- Docker on an internet-connected environment.

To export a TAR file of the image, run:

```shell
docker image save -o grafana-image-renderer.tar grafana/grafana-image-renderer:latest
```

If you're using a different CPU architecture than the air-gapped environment,
you may need to specify `--platform` when saving the image. For example, if you
have an air-gapped x86_64 (amd64) machine, use `--platform linux/amd64`.

Next, transfer the file to the machine.

Finally, import the image on the air-gapped environment:

```shell
docker image load grafana-image-renderer.tar
```

### Without Docker

You will need:

- A way to transfer data into the environment, such as a USB stick, SD card,
  external hard-drive, or similar.

We release binary files for Linux and Windows on our [GitHub Releases][releases]
page. Download the appropriate binary for your system, and transfer it to the
machine.

You will also need to install a Chromium-based browser separately.

[releases]: https://github.com/grafana/grafana-image-renderer/releases

## Using Docker without Grafana being dockerised

You can use [host
networking](https://docs.docker.com/engine/network/tutorials/host/) instead,
or the binary releases.

## Using the image renderer service on Windows without Docker

You can download the Windows binaries from the [GitHub Releases page][releases].
For example, to use the image renderer service with the Brave browser on an
ARM64 Windows host, run:

```powershell
.\grafana-image-renderer-windows-arm64.exe server --browser.path "C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe"
```

The browser must be installed separately and be a Chromium-based browser.

## Custom CA certificates in Chromium

Identifying that the CA certificate is the problem is done by checking for
`net::ERR_CERT_AUTHORITY_INVALID` errors in the service's logs. Finding these
errors may require enabling debug logging, via the `--log.level debug` option.

### Non-containerized Linux

For non-containerized Linux, you need `nss` tools (`libnss3-tools` on Debian).
You will also need to know the `$HOME` directory of the user running the
service, which can be found via running either `eval echo ~username` (for
example: `eval echo ~grafana`), or via `getent passwd username` (for example:
`getent passwd grafana`). Run the following for that user:

```shell
certutil -d sql:"$HOME"/.pki/nssdb -A -n internal-root-ca -t C -i /path/to/internal-root-ca-here.crt.pem
```

You might also need other tooling. The error message will likely indicate what
is missing from your environment.

### Non-containerized Windows

For non-containerized, you need to do the same as on Linux,
but to your global store:

```powershell
certutil –addstore "Root" <path>/internal-root-ca-here.crt.pem
```

### Container

The easiest way is to integrate the CA certificate directly into your own Docker
image, based on the official one:

```dockerfile
# Consider using a pinned version.
FROM grafana/grafana-image-renderer:latest

# Elevate our permissions to access system resources.
USER root

RUN mkdir -p /usr/local/share/ca-certificates/
# Convert from .pem to .crt
RUN openssl x509 -inform PEM -in rootCA.pem -out /usr/local/share/ca-certificates/rootCA.crt

# Regenerate the CA certificates in the container.
RUN update-ca-certificates --fresh

# Reassume the nonroot user for the service execution.
USER nonroot
# Note: for Kubernetes, OpenShift, and other setups, this may need a numeric ID. See the upstream Dockerfile for which UID to use.

# Some CA certificates also need to explicitly be included in the user's network security services database.
RUN mkdir -p /home/nonroot/.pki/nssdb
RUN certutil -d sql:/home/nonroot/.pki/nssdb -A -n internal-root-ca -t C -i /usr/local/share/ca-certificates/rootCA.crt
```

## Distorted panels in the PDF export

This is most commonly caused by using the old PDF rendering engine in Grafana.
To identify whether this applies to you, check whether the `newPDFRendering`
feature flag has been explicitly set to `false` in Grafana's configuration.

To solve this problem, remove the feature toggle override.
