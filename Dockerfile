FROM debian:12-slim@sha256:7e490910eea2861b9664577a96b54ce68ea3e02ce7f51d89cb0103a6f9c386e0 AS debian-updated

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

# If we ever need to bust the cache, just change the date here.
# While we don't cache anything in Drone, that might not be true when we migrate to GitHub Actions where some action might automatically enable layer caching.
# This is fine, but is terrible in situations where we want to _force_ an update of a package.
RUN echo 'cachebuster 2025-10-13' && apt-get update

FROM debian-updated AS debs

ARG CHROMIUM_VERSION=141.0.7390.65
RUN apt-cache depends chromium=${CHROMIUM_VERSION} chromium-driver chromium-shell chromium-sandbox font-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst fonts-freefont-ttf libxss1 unifont fonts-open-sans fonts-roboto fonts-inter bash util-linux openssl tini ca-certificates locales libnss3-tools \
  --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
  find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

FROM debian:testing-slim@sha256:21c02c216f921ccd2cd1a95cd64499c52e5699c6b04029b5ad4ca68f0b6f1121 AS ca-certs

RUN apt-get update
RUN apt-cache depends ca-certificates \
  --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
  find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

# While we can't move to Debian 13 yet for the final image, use its new build of busybox with security fixes.
FROM debian:13-slim@sha256:1caf1c703c8f7e15dcf2e7769b35000c764e6f50e4d7401c355fb0248f3ddfdb AS busybox

RUN apt-get update
RUN apt-cache depends busybox-static \
  --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
  find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

FROM node:22-alpine@sha256:dbcedd8aeab47fbc0f4dd4bffa55b7c3c729a707875968d467aaaea42d6225af AS build

WORKDIR /src
COPY . ./

RUN yarn install --pure-lockfile
RUN yarn run build
RUN rm -rf node_modules/ && yarn install --pure-lockfile --production

FROM gcr.io/distroless/nodejs22-debian12:nonroot@sha256:e5403162edcafbf01af6411d6a94858dfde55de6f3e9c821b1c490379c4bbac2

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

COPY --from=debs /dpkg /
COPY --from=busybox /dpkg/usr/bin/busybox /bin/busybox
COPY --from=busybox /dpkg/usr/bin/busybox /usr/bin/busybox
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
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD="true"
ENV NODE_ENV=production
ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8

COPY --from=build /src/node_modules node_modules
COPY --from=build /src/build build
COPY --from=build /src/proto proto
COPY --from=build /src/default.json config.json
COPY --from=build /src/plugin.json plugin.json

USER root

RUN chgrp -R 0 /home/nonroot && chmod -R g=u /home/nonroot

USER 65532

EXPOSE 8081

ENTRYPOINT ["tini", "--", "/nodejs/bin/node"]
CMD ["build/app.js", "server", "--config=config.json"]
HEALTHCHECK --interval=10s --retries=3 --timeout=3s \
  CMD ["wget", "-O-", "-q", "http://localhost:8081/"]
