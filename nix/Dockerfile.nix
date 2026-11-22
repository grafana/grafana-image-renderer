ARG NIX_BUILDER_IMAGE=nixos/nix:2.35.3@sha256:ff480dbb23511aa1c10710aa068ceea5b93fed9bfce5f5379f3a6b07282cbc7b
FROM ${NIX_BUILDER_IMAGE} AS nix

FROM nix AS source
RUN printf 'experimental-features = nix-command flakes\nsandbox = false\nfilter-syscalls = false\n' > /etc/renderer-nix.conf
ENV NIX_USER_CONF_FILES=/etc/renderer-nix.conf
WORKDIR /src
COPY . .

FROM source AS chromium-info-build
ARG TARGETARCH
ARG NIX_BUILDER_IMAGE
RUN --mount=type=cache,id=image-renderer-nix-${TARGETARCH}-${NIX_BUILDER_IMAGE},target=/nix,from=nix,source=/nix,sharing=locked \
    set -eu; \
    mkdir -p /out; \
    nix run --no-update-lock-file .#update-chromium \
        -- /out/chromium-info.json

FROM scratch AS chromium-info
COPY --from=chromium-info-build /out/chromium-info.json /chromium-info.json

FROM source AS build
ARG TARGETARCH
ARG NIX_BUILDER_IMAGE
RUN --mount=type=cache,id=image-renderer-nix-${TARGETARCH}-${NIX_BUILDER_IMAGE},target=/nix,from=nix,source=/nix,sharing=locked \
    set -eu; \
    archive="$(nix build .#image --no-update-lock-file --no-link --print-out-paths)"; \
    mkdir -p /out; \
    cp "$archive" /out/image.tar.gz

FROM scratch AS archive
COPY --from=build /out/image.tar.gz /image.tar.gz
