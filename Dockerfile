FROM debian:12-slim AS debian-updated

# If we ever need to bust the cache, just change the date here.
# While we don't cache anything in Drone, that might not be true when we migrate to GitHub Actions where some action might automatically enable layer caching.
# This is fine, but is terrible in situations where we want to _force_ an update of a package.
RUN echo 'cachebuster 2025-07-16' && apt-get update

FROM debian-updated AS chromium

# This downloads the Chromium components we need, along with some fonts as is necessary.

RUN apt-get install -y wget gnupg && \
    wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | gpg --dearmor -o /usr/share/keyrings/googlechrome-linux-keyring.gpg && \
    sh -c 'echo "deb [arch=amd64 signed-by=/usr/share/keyrings/googlechrome-linux-keyring.gpg] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list' && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        google-chrome-stable fonts-ipafont-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst fonts-freefont-ttf libxss1 unifont fonts-open-sans fonts-roboto fonts-inter
RUN dpkg-reconfigure fontconfig fontconfig-config
RUN fc-match -s Inter | grep 'Inter'

RUN apt-get install -y --no-install-recommends gawk
RUN mkdir -p /chrome-libs/ && \
    ldd /opt/google/chrome/chrome /bin/bash | grep '=> /lib/' | awk 'BEGIN{RS=" \\(";FS=" => ";}NF>1{ print $NF; }' | grep '.so' | xargs -I'{}' sh -c 'mkdir -p /chrome-libs/$(dirname {}) && cp {} /chrome-libs/$(dirname {})/'

FROM gcr.io/distroless/nodejs22-debian12:debug AS chromium-libs
SHELL ["/busybox/sh", "-c"]

COPY --from=chromium /chrome-libs /chrome-libs

# We only want the libs that Chromium needs and we don't already provide to be copied into the distroless image.
# We ultimately trust the distroless image _more_ on the dependencies it provides.
WORKDIR /chrome-libs/
RUN find . -type f -exec sh -c '[ -f "$(echo "$1" | cut -c2- | xargs)" ] && rm "$1"' sh '{}' \;

FROM node:22-alpine AS build

WORKDIR /src
COPY . ./

RUN yarn install --pure-lockfile
RUN yarn run build
RUN rm -rf node_modules/ && yarn install --pure-lockfile --production

FROM gcr.io/distroless/nodejs22-debian12:debug-nonroot

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

COPY --from=chromium /etc/fonts /etc/fonts
COPY --from=chromium /usr/share/fonts /usr/share/fonts
COPY --from=chromium /usr/share/fontconfig /usr/share/fontconfig
COPY --from=chromium /usr/share/xml/fontconfig /usr/share/xml/fontconfig
COPY --from=chromium /var/cache/fontconfig /var/cache/fontconfig
COPY --from=chromium-libs /chrome-libs/lib /lib
COPY --from=chromium /opt/google/chrome /opt/google/chrome

COPY --from=chromium /bin/bash /bin/bash

ENV CHROME_BIN="/opt/google/chrome/google-chrome"
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD="true"
ENV NODE_ENV=production

COPY --from=build /src/node_modules node_modules
COPY --from=build /src/build build
COPY --from=build /src/proto proto
COPY --from=build /src/default.json config.json
COPY --from=build /src/plugin.json plugin.json

EXPOSE 8081

CMD ["build/app.js", "server", "--config=config.json"]
