# This Dockerfile does two things in parallel:
#   1. It builds a statically linked Go binary. This is fine to use in Debian, Alpine, RHEL, etc. base-images.
#        -> Why static linking? We want to ensure we can switch base-image with little to no effort.
#   2. It builds a running environment. This is the environment that exists for the Go binary, and should have all necessary pieces to run the application.

FROM golang:1.24.6-alpine AS app

RUN apk add --no-cache git

WORKDIR /src
COPY . ./

RUN --mount=type=cache,target=/go/pkg/mod CGO_ENABLED=0 go build \
    -o grafana-image-renderer \
    -buildvcs \
    -ldflags '-s -w -extldflags "-static"' \
    .

FROM debian:12-slim@sha256:b1a741487078b369e78119849663d7f1a5341ef2768798f7b7406c4240f86aef AS debs

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

# If we ever need to bust the cache, just change the date here.
# While we don't cache anything in Drone, that might not be true when we migrate to GitHub Actions where some action might automatically enable layer caching.
# This is fine, but is terrible in situations where we want to _force_ an update of a package.
RUN echo 'cachebuster 2025-10-06' && apt-get update

ARG CHROMIUM_VERSION=141.0.7390.54
RUN apt-cache depends chromium=${CHROMIUM_VERSION} chromium-driver chromium-shell chromium-sandbox font-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst fonts-freefont-ttf libxss1 unifont fonts-open-sans fonts-roboto fonts-inter bash busybox util-linux openssl tini ca-certificates locales libnss3-tools \
    --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
    find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

FROM debian:testing-slim@sha256:12ce5b90ca703a11ebaae907649af9b000e616f49199a2115340e0cdf007e42a AS ca-certs

RUN apt-get update
RUN apt-cache depends ca-certificates \
    --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
    find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

FROM gcr.io/distroless/base-debian12:nonroot AS output_image

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/go.Dockerfile"

COPY --from=debs /dpkg /
COPY --from=ca-certs /dpkg/usr/share/ca-certificates /usr/share/ca-certificates

USER root
SHELL ["/bin/busybox", "sh", "-c"]
RUN /bin/busybox --install
# Verify that the browser was actually installed.
RUN /usr/bin/chromium --version
# This is so the browser can write file names that contain non-ASCII characters.
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen en_US.UTF-8
RUN fc-cache -fr
RUN update-ca-certificates --fresh
USER nonroot

ENV CHROME_BIN="/usr/bin/chromium"
ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8

USER root
RUN chgrp -R 0 /home/nonroot && chmod -R g=u /home/nonroot
COPY --from=app /src/grafana-image-renderer /usr/bin/grafana-image-renderer
USER 65532
EXPOSE 8081

ENTRYPOINT ["tini", "--", "/usr/bin/grafana-image-renderer"]
CMD ["server"]
HEALTHCHECK --interval=10s --retries=3 --timeout=3s --start-interval=250ms --start-period=30s \
    CMD ["/usr/bin/grafana-image-renderer", "healthcheck"]
