# Base stage
FROM node:18-alpine AS base

ENV CHROME_BIN="/usr/bin/chromium-browser"
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD="true"

WORKDIR /usr/src/app

RUN apk --no-cache upgrade && \
    apk add --no-cache udev ttf-opensans unifont chromium ca-certificates dumb-init && \
    rm -rf /tmp/*

# Build stage
FROM base AS build

COPY . ./

RUN yarn install --pure-lockfile
RUN yarn run build

# Production dependencies stage
FROM base AS prod-dependencies

COPY package.json yarn.lock ./
RUN yarn install --pure-lockfile --production

# Final stage
FROM base

LABEL maintainer="Grafana team <hello@grafana.com>"

ARG GF_UID="472"
ARG GF_GID="472"
ENV GF_PATHS_HOME="/usr/src/app"

WORKDIR $GF_PATHS_HOME

RUN addgroup -S -g $GF_GID grafana && \
    adduser -S -u $GF_UID -G grafana grafana && \
    mkdir -p "$GF_PATHS_HOME" && \
    chown -R grafana:grafana "$GF_PATHS_HOME"

ENV NODE_ENV=production

COPY --from=prod-dependencies /usr/src/app/node_modules node_modules
COPY --from=build /usr/src/app/build build
COPY --from=build /usr/src/app/proto proto
COPY --from=build /usr/src/app/default.json config.json
COPY --from=build /usr/src/app/plugin.json plugin.json

EXPOSE 8081

USER grafana

ENTRYPOINT ["dumb-init", "--"]
CMD ["node", "build/app.js", "server", "--config=config.json"]
