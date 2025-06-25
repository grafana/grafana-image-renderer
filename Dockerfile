# Base stage
FROM node:20-alpine AS base

ENV CHROME_BIN="/usr/bin/chromium-browser"
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD="true"

# Folder used by puppeteer to write temporal files
ENV XDG_CONFIG_HOME=/tmp/.chromium
ENV XDG_CACHE_HOME=/tmp/.chromium

WORKDIR /usr/src/app

# We use edge for Chromium to get the latest release.
# Can be back to stable when 3.22 gets .103.
RUN apk --no-cache upgrade && \
    apk add --no-cache udev ttf-opensans unifont ca-certificates dumb-init && \
    apk add --no-cache 'chromium>=137.0.7151.119' 'chromium-swiftshader>=137.0.7151.119' --repository=https://dl-cdn.alpinelinux.org/alpine/edge/community && \
    # Remove NPM-related files and directories
    rm -rf /usr/local/lib/node_modules/npm && \
    rm -rf /usr/local/bin/npm && \
    rm -rf /usr/local/bin/npx && \
    rm -rf /root/.npm && \
    rm -rf /root/.node-gyp && \
    # Clean up
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
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana-image-renderer/tree/master/Dockerfile"

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
