FROM node:22-trixie AS build

WORKDIR /src
COPY . ./

RUN yarn install --pure-lockfile
RUN yarn run build
RUN rm -rf node_modules/ && yarn install --pure-lockfile --production

FROM node:22-trixie AS output_image

LABEL maintainer="Grafana team <hello@grafana.com>"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

# If we ever need to bust the cache, just change the date here.
RUN echo 'cachebuster 2025-10-17' && apt-get update

RUN apt-get install -y --no-install-recommends --no-install-suggests \
  bash util-linux openssl tini ca-certificates locales libnss3-tools ca-certificates

ARG CHROMIUM_VERSION=141.0.7390.107
RUN apt-get satisfy -y --no-install-recommends --no-install-suggests \
  "chromium (>=${CHROMIUM_VERSION}), chromium-driver (>=${CHROMIUM_VERSION}), chromium-shell (>=${CHROMIUM_VERSION}), chromium-sandbox (>=${CHROMIUM_VERSION})"
RUN apt-get clean && rm -rf /var/lib/apt/lists/*

# This is so the browser can write file names that contain non-ASCII characters.
RUN echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && locale-gen en_US.UTF-8
RUN fc-cache -fr
RUN update-ca-certificates --fresh
RUN useradd --create-home --system --uid 65532 --user-group nonroot
USER 65532

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

ENTRYPOINT ["tini", "--", "node"]
CMD ["build/app.js", "server", "--config=config.json"]
HEALTHCHECK --interval=10s --retries=3 --timeout=3s \
  CMD ["wget", "-O-", "-q", "http://localhost:8081/"]
