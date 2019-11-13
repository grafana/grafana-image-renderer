FROM node:10-buster-slim AS base

RUN apt-get update \
    && apt-get install -y chromium fonts-open-sans ca-certificates --no-install-recommends \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /src/*.deb

ADD https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 /usr/local/bin/dumb-init
RUN chmod +x /usr/local/bin/dumb-init


FROM base as build

ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD="true"

WORKDIR /usr/src/app

COPY . .

RUN yarn install --pure-lockfile
RUN yarn run build


FROM base

ENV CHROME_BIN="/usr/bin/chromium"

COPY --from=build /usr/src/app/node_modules node_modules
COPY --from=build /usr/src/app/build build
COPY --from=build /usr/src/app/proto proto

RUN groupadd --system chrome && \
    useradd --system --create-home --gid chrome --groups audio,video chrome && \
    mkdir --parents /home/chrome/reports && \
    chown --recursive chrome:chrome /home/chrome

USER chrome

EXPOSE 8081

ENTRYPOINT ["dumb-init", "--"]

CMD ["node", "build/app.js", "server", "--port=8081"]

