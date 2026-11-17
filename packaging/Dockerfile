# Install authenticated Debian assets; export only the browser, fonts, and two
# library ABIs unavailable in Wolfi. All other runtime libraries come from Wolfi.
FROM debian:trixie-20261104@sha256:9c6e381421d0f48d1b8133e573f33c185ee16cbf505eda9c4a8a4791b25ea8a6 AS download

ARG CHROMIUM_VERSION
COPY packaging/debian-sbom.jq /tmp/debian-sbom.jq

RUN --mount=type=tmpfs,target=/var/lib/apt/lists \
    apt-get update && \
    apt-get install -y --no-install-recommends --no-install-suggests \
      "chromium=${CHROMIUM_VERSION}" "chromium-common=${CHROMIUM_VERSION}" "chromium-sandbox=${CHROMIUM_VERSION}" jq \
      fonts-ipaexfont-gothic fonts-wqy-zenhei fonts-thai-tlwg fonts-khmeros fonts-kacst-one fonts-freefont-ttf \
      unifont fonts-open-sans fonts-roboto fonts-inter fonts-recommended && \
    mkdir -p /sources/usr/lib /sources/usr/share/grafana-image-renderer && \
    cp -a /usr/lib/chromium /sources/usr/lib/ && \
    cp -L /usr/lib/*-linux-gnu/libjpeg.so.62 /usr/lib/*-linux-gnu/libXNVCtrl.so.0 /sources/usr/lib/ && \
    cp -a /usr/share/fonts /usr/share/common-licenses /sources/usr/share/ && \
    dpkg-query -W -f '${db:Status-Status} ${Package}\n' \
      'fonts-*' unifont chromium chromium-common chromium-sandbox libjpeg62-turbo libxnvctrl0 | \
      while read -r status package; do \
        if [ "$status" = installed ]; then \
          install -D "/usr/share/doc/$package/copyright" "/sources/licenses/$package/copyright"; \
        fi; \
      done && \
    dpkg-query -W -f '${Package}\t${Version}\t${Architecture}\t${source:Package}\t${source:Version}\n' \
      chromium chromium-common chromium-sandbox libjpeg62-turbo libxnvctrl0 | \
      jq -Rn --arg created "$(date -u +%Y-%m-%dT%H:%M:%SZ)" --arg arch "$(dpkg --print-architecture)" -f /tmp/debian-sbom.jq \
        > /sources/usr/share/grafana-image-renderer/debian.spdx.json

FROM scratch
COPY --from=download /sources/ /
