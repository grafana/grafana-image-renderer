FROM debian:12-slim@sha256:df52e55e3361a81ac1bead266f3373ee55d29aa50cf0975d440c2be3483d8ed3 AS debian-updated

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

# If we ever need to bust the cache, just change the date here.
# While we don't cache anything in Drone, that might not be true when we migrate to GitHub Actions where some action might automatically enable layer caching.
# This is fine, but is terrible in situations where we want to _force_ an update of a package.
RUN echo 'cachebuster 2025-09-11' && apt-get update

FROM debian-updated AS debs

ARG CHROMIUM_VERSION=140.0.7339.127
RUN apt-cache depends chromium=${CHROMIUM_VERSION} chromium-driver chromium-shell chromium-sandbox font-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst fonts-freefont-ttf libxss1 unifont fonts-open-sans fonts-roboto fonts-inter bash util-linux openssl tini ca-certificates locales libnss3-tools \
    --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
    find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

FROM debian:testing-slim@sha256:f7e3f921f98e1a51742ceda3cbe52ac815e60e83b0c381ea9444ac17a1e32cb4 AS ca-certs

RUN apt-get update
RUN apt-cache depends ca-certificates \
    --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
    find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

# While we can't move to Debian 13 yet for the final image, use its new build of busybox with security fixes.
FROM debian:13-slim@sha256:c2880112cc5c61e1200c26f106e4123627b49726375eb5846313da9cca117337 AS busybox

RUN apt-get update
RUN apt-cache depends busybox-static \
    --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
    find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

FROM node:22-alpine@sha256:d2166de198f26e17e5a442f537754dd616ab069c47cc57b889310a717e0abbf9 AS build

WORKDIR /src
COPY . ./

RUN yarn install --pure-lockfile
RUN yarn run build
RUN rm -rf node_modules/ && yarn install --pure-lockfile --production

FROM gcr.io/distroless/nodejs22-debian12:nonroot@sha256:82f784c7f478cf5129d5446f99b442bbc17ef8ab48fbea25c58e05b2859896f7

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
