FROM debian:12-slim AS debian-updated

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

# If we ever need to bust the cache, just change the date here.
# While we don't cache anything in Drone, that might not be true when we migrate to GitHub Actions where some action might automatically enable layer caching.
# This is fine, but is terrible in situations where we want to _force_ an update of a package.
RUN echo 'cachebuster 2025-07-16' && apt-get update

FROM debian-updated AS debs

RUN apt-cache depends chromium chromium-driver chromium-shell chromium-sandbox font-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst fonts-freefont-ttf libxss1 unifont fonts-open-sans fonts-roboto fonts-inter bash busybox util-linux openssl \
    --recurse --no-recommends --no-suggests --no-conflicts --no-breaks --no-replaces --no-enhances --no-pre-depends | grep '^\w' | xargs apt-get download
RUN mkdir /dpkg && \
    find . -type f -name '*.deb' -exec sh -c 'dpkg --extract "$1" /dpkg || exit 5' sh '{}' \;

FROM debian:testing-slim AS ca-certs

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates
RUN update-ca-certificates --fresh

FROM node:22-alpine AS build

WORKDIR /src
COPY . ./

RUN yarn install --pure-lockfile
RUN yarn run build
RUN rm -rf node_modules/ && yarn install --pure-lockfile --production

FROM gcr.io/distroless/nodejs22-debian12:nonroot

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

COPY --from=debs /dpkg /
COPY --from=ca-certs /etc/ssl/certs /etc/ssl/certs

USER root
SHELL ["/bin/busybox", "sh", "-c"]
RUN /bin/busybox --install
# Verify that the browser was actually installed.
RUN /usr/bin/chromium --version
RUN fc-cache -fr
USER nonroot

ENV CHROME_BIN="/usr/bin/chromium"
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD="true"
ENV NODE_ENV=production

COPY --from=build /src/node_modules node_modules
COPY --from=build /src/build build
COPY --from=build /src/proto proto
COPY --from=build /src/default.json config.json
COPY --from=build /src/plugin.json plugin.json

EXPOSE 8081

CMD ["build/app.js", "server", "--config=config.json"]
