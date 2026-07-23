# This Dockerfile does two things in parallel:
#   1. It builds a statically linked Go binary. This is fine to use in Debian, Alpine, RHEL, etc. base-images.
#        -> Why static linking? We want to ensure we can switch base-image with little to no effort.
#   2. It builds a running environment. This is the environment that exists for the Go binary, and should have all necessary pieces to run the application.
#
# Two runtime variants are produced from this file:
#   - output_debian (default)
#   - output_alpine

# renovate: depName=chromium
ARG CHROMIUM_VERSION=150.0.7871.114
# If we ever need to bust the package cache, just change the date here.
ARG CACHE_BUSTER_DATE=2026-07-15

FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS app

RUN apk add --no-cache git

WORKDIR /src
COPY . ./

RUN --mount=type=cache,target=/go/pkg/mod CGO_ENABLED=0 go build \
  -o grafana-image-renderer \
  -buildvcs \
  -ldflags '-s -w -extldflags "-static"' \
  .

FROM debian:trixie-20260713@sha256:fac46bff2e02f51425b6e33b0e1169f55dfb053d83511ca28aa50c09fd5ed7a4 AS output_debian

ARG CHROMIUM_VERSION
ARG CACHE_BUSTER_DATE

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

RUN echo "cachebuster ${CACHE_BUSTER_DATE}" && apt-get update && apt-get upgrade -y --no-install-recommends --no-install-suggests

RUN apt-get install -y --no-install-recommends --no-install-suggests \
  fonts-ipaexfont-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst-one fonts-freefont-ttf \
  libxss1 unifont fonts-open-sans fonts-roboto fonts-inter fonts-recommended \
  bash util-linux openssl tini ca-certificates locales libnss3-tools ca-certificates

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

# renovate: datasource=docker depName=alpine
FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS output_alpine

ARG CHROMIUM_VERSION
ARG CACHE_BUSTER_DATE

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

RUN echo "cachebuster ${CACHE_BUSTER_DATE}" && apk upgrade --no-cache

RUN apk add --no-cache \
  "chromium>=${CHROMIUM_VERSION}" \
  "chromium-swiftshader>=${CHROMIUM_VERSION}" \
  tini ca-certificates openssl nss-tools \
  font-noto-cjk font-noto-thai font-noto-khmer font-noto-arabic font-noto-emoji \
  font-opensans font-roboto font-inter font-urw-base35 font-dejavu font-unifont

# Alpine packages the URW fontconfig aliases without enabling them. Enable the
# packaged rules so Helvetica resolves to Nimbus Sans, as it does on Debian.
RUN ln -s /usr/share/fontconfig/conf.default/69-urw-*.conf /etc/fonts/conf.d/

RUN fc-cache -fr
RUN update-ca-certificates --fresh

RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S -G nonroot -h /home/nonroot nonroot
RUN chgrp -R 0 /home/nonroot && chmod -R g=u /home/nonroot
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

# Keep `docker build .` backwards-compatible with the Debian image.
FROM output_debian AS output_image
