# This Dockerfile does two things in parallel:
#   1. It builds a statically linked Go binary. This is fine to use in Debian, Alpine, RHEL, etc. base-images.
#        -> Why static linking? We want to ensure we can switch base-image with little to no effort.
#   2. It builds a running environment. This is the environment that exists for the Go binary, and should have all necessary pieces to run the application.

FROM golang:1.26.4-alpine@sha256:f23e8b227fb4493eabe03bede4d5a32d04092da71962f1fb79b5f7d1e6c2a17f AS app

RUN apk add --no-cache git

WORKDIR /src
COPY . ./

RUN --mount=type=cache,target=/go/pkg/mod CGO_ENABLED=0 go build \
  -o grafana-image-renderer \
  -buildvcs \
  -ldflags '-s -w -extldflags "-static"' \
  .

FROM debian:trixie-20260518@sha256:4ae67669760b807c19f23902a3fd7c121a6a70cf2ae709035674b23e712e4d62 AS output_image

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

# If we ever need to bust the cache, just change the date here.
RUN echo 'cachebuster 2026-05-18' && apt-get update && apt-get upgrade -y --no-install-recommends --no-install-suggests

RUN apt-get install -y --no-install-recommends --no-install-suggests \
  fonts-ipaexfont-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst-one fonts-freefont-ttf \
  libxss1 unifont fonts-open-sans fonts-roboto fonts-inter fonts-recommended \
  bash util-linux openssl tini ca-certificates locales libnss3-tools ca-certificates

# renovate: depName=chromium
ARG CHROMIUM_VERSION=148.0.7778.167
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
