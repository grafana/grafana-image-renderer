FROM nixos/nix:2.35.3@sha256:ff480dbb23511aa1c10710aa068ceea5b93fed9bfce5f5379f3a6b07282cbc7b AS nix

FROM nix AS build
RUN printf 'experimental-features = nix-command flakes\nsandbox = false\nfilter-syscalls = false\n' > /etc/renderer-nix.conf
ENV NIX_USER_CONF_FILES=/etc/renderer-nix.conf
WORKDIR /src
COPY . .

RUN --mount=type=cache,id=image-renderer-nix,target=/nix,from=nix,source=/nix,sharing=locked \
    set -eu; \
    archive="$(nix build .#image --no-update-lock-file --no-link --print-out-paths)"; \
    mkdir -p /out; \
    cp "$archive" /out/image.tar.gz

FROM scratch AS archive
COPY --from=build /out/image.tar.gz /image.tar.gz
