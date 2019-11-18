FROM node:10-alpine AS base

ENV CHROME_BIN="/usr/bin/chromium-browser"
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD="true"

WORKDIR /usr/src/app

RUN \
  echo "http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories \
  && echo "http://dl-cdn.alpinelinux.org/alpine/edge/main" >> /etc/apk/repositories \
  && echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories \
  && apk --no-cache  update \
  && apk --no-cache  upgrade \
  && apk add --no-cache --virtual .build-deps \
    udev ttf-opensans chromium \
		ca-certificates dumb-init curl nghttp2 openldap \
  && rm -rf /var/cache/apk/* /tmp/*

FROM base as build

COPY . ./

RUN yarn install --pure-lockfile
RUN yarn run build

EXPOSE 8081

CMD [ "yarn", "run", "dev" ]

FROM base

COPY --from=build /usr/src/app/node_modules node_modules
COPY --from=build /usr/src/app/build build
COPY --from=build /usr/src/app/proto proto

EXPOSE 8081

ENTRYPOINT ["dumb-init", "--"]

CMD ["node", "build/app.js", "server", "--port=8081"]