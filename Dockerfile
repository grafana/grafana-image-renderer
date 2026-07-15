# This Dockerfile builds two things in parallel, sharing as many layers as possible:
#   1. A statically linked Go binary. This is fine to use in Debian, Alpine, RHEL, etc. base-images.
#        -> Why static linking? We want to ensure we can switch base-image with little to no effort.
#   2. A running environment for that Go binary, produced in two variants that share a common Debian layer
#      (`runtime_base`) so the expensive Chromium install happens exactly once:
#        - `output_image`            -> the regular Debian-based image (default `docker build .` target).
#        - `distroless_output_image` -> a distroless (base-debian13) image.

FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS app

RUN apk add --no-cache git

WORKDIR /src
COPY . ./

RUN --mount=type=cache,target=/go/pkg/mod CGO_ENABLED=0 go build \
  -o grafana-image-renderer \
  -buildvcs \
  -ldflags '-s -w -extldflags "-static"' \
  .

# runtime_base holds the shared Debian runtime environment: Chromium, its dependencies, fonts and the
# handful of tools the acceptance tests exercise, with all the package post-install steps (locale, font
# cache, CA certificates, the nonroot user) already applied.
FROM debian:trixie-20260713@sha256:fac46bff2e02f51425b6e33b0e1169f55dfb053d83511ca28aa50c09fd5ed7a4 AS runtime_base

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

# If we ever need to bust the cache, just change the date here.
RUN echo 'cachebuster 2026-07-14' && apt-get update && apt-get upgrade -y --no-install-recommends --no-install-suggests

RUN apt-get install -y --no-install-recommends --no-install-suggests \
  fonts-ipaexfont-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst-one fonts-freefont-ttf \
  libxss1 unifont fonts-open-sans fonts-roboto fonts-inter fonts-recommended \
  bash util-linux openssl tini ca-certificates locales libnss3-tools ca-certificates

# renovate: depName=chromium
ARG CHROMIUM_VERSION=150.0.7871.114
RUN apt-get satisfy -y --no-install-recommends --no-install-suggests \
  "chromium (>=${CHROMIUM_VERSION}), chromium-driver (>=${CHROMIUM_VERSION}), chromium-shell (>=${CHROMIUM_VERSION}), chromium-sandbox (>=${CHROMIUM_VERSION})"

# There is no point to us shipping headers.
RUN dpkg -l | grep -E -- '-dev|-headers' | awk '{ print $2; }' | xargs apt-get remove -y
# Do a final automatic clean-up.
RUN apt-get autoremove -y

RUN apt-get clean && rm -rf /var/lib/apt/lists/*

# This is so the browser can write file names that contain non-ASCII characters.
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen en_US.UTF-8
RUN fc-cache -fr
RUN update-ca-certificates --fresh

RUN useradd --create-home --system --uid 65532 --user-group nonroot
RUN chgrp -R 0 /home/nonroot && chmod -R g=u /home/nonroot

# busybox provides a static shell + coreutils for the distroless variant, which ships no shell of its own.
FROM debian:trixie-20260713@sha256:fac46bff2e02f51425b6e33b0e1169f55dfb053d83511ca28aa50c09fd5ed7a4 AS busybox

RUN apt-get update && apt-get install -y --no-install-recommends --no-install-suggests busybox-static

# distroless_output_image is the "distroless" variant. It is deliberately NOT the last stage.
FROM gcr.io/distroless/base-debian13:nonroot AS distroless_output_image

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

COPY --from=busybox /bin/busybox /usr/bin/busybox

# Copy everything from the base to run Chromium properly.
# - Shared libraries: glibc, Chromium's own libs (/usr/lib/chromium), NSS, fontconfig, OpenSSL and the generated locale archive.
# - The binaries the application and the acceptance tests invoke by name (findmnt is used by the chromium launcher's /etc/chromium.d/dev-shm hook).
# - Configuration + data the tools above need at runtime: chromium launcher config, fonts + fontconfig cache, CA certificates, and the time-zone database.
# - Home directory, pre-configured for arbitrary (e.g. OpenShift) UIDs via the root group.
COPY --from=runtime_base /usr/lib/ /usr/lib/
COPY --from=runtime_base /usr/bin/chromium /usr/bin/openssl /usr/bin/certutil /usr/bin/tini /usr/bin/findmnt /usr/bin/
COPY --from=runtime_base /usr/bin/fc-match /usr/bin/fc-cache /usr/bin/fc-list /usr/bin/
COPY --from=runtime_base /usr/sbin/update-ca-certificates /usr/sbin/update-ca-certificates
COPY --from=runtime_base /etc/chromium.d/ /etc/chromium.d/
COPY --from=runtime_base /etc/fonts/ /etc/fonts/
COPY --from=runtime_base /usr/share/fonts/ /usr/share/fonts/
COPY --from=runtime_base /usr/share/fontconfig/ /usr/share/fontconfig/
COPY --from=runtime_base /var/cache/fontconfig/ /var/cache/fontconfig/
COPY --from=runtime_base /etc/ssl/ /etc/ssl/
COPY --from=runtime_base /usr/share/ca-certificates/ /usr/share/ca-certificates/
COPY --from=runtime_base /usr/share/zoneinfo/ /usr/share/zoneinfo/
COPY --from=runtime_base /home/nonroot/ /home/nonroot/

COPY --from=app /src/grafana-image-renderer /usr/bin/grafana-image-renderer

# Create the busybox applet symlinks. It also needs to run as root to write into the bin directories.
USER root
SHELL ["/usr/bin/busybox", "sh", "-c"]
RUN /usr/bin/busybox --install

WORKDIR /home/nonroot
# The base image already ships `nonroot` (uid 65532) in /etc/passwd, but we set the UID numerically so
# tooling that reads the image config (e.g. Kubernetes runAsNonRoot) sees a numeric user.
USER 65532

EXPOSE 8081

ENV CHROME_BIN="/usr/bin/chromium"
ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8
ENTRYPOINT ["tini", "--", "/usr/bin/grafana-image-renderer"]
CMD ["server"]
HEALTHCHECK --interval=10s --retries=3 --timeout=3s --start-interval=250ms --start-period=30s \
  CMD ["/usr/bin/grafana-image-renderer", "healthcheck"]

# output_image is the regular Debian-based image. It is the LAST stage on purpose, so that a plain
# `docker build .` (used by CI and by most consumers) keeps producing it by default.
FROM runtime_base AS output_image

WORKDIR /home/nonroot
USER 65532

COPY --from=app /src/grafana-image-renderer /usr/bin/grafana-image-renderer

EXPOSE 8081

ENV CHROME_BIN="/usr/bin/chromium"
ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8
ENTRYPOINT ["tini", "--", "/usr/bin/grafana-image-renderer"]
CMD ["server"]
HEALTHCHECK --interval=10s --retries=3 --timeout=3s --start-interval=250ms --start-period=30s \
  CMD ["/usr/bin/grafana-image-renderer", "healthcheck"]
