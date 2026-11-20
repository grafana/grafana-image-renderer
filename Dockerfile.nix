ARG NIX_BUILDER_IMAGE=nixos/nix:2.35.3@sha256:ff480dbb23511aa1c10710aa068ceea5b93fed9bfce5f5379f3a6b07282cbc7b
FROM ${NIX_BUILDER_IMAGE} AS nix

FROM nix AS chromium-info-build
WORKDIR /src
COPY . .

ARG TARGETARCH
ARG NIX_BUILDER_IMAGE
RUN --mount=type=cache,id=image-renderer-nix-${TARGETARCH}-${NIX_BUILDER_IMAGE},target=/nix,from=nix,source=/nix,sharing=locked \
    set -eu; \
    case "$TARGETARCH" in \
        arm64) system=aarch64-linux ;; \
        amd64) system=x86_64-linux ;; \
        *) printf 'Unsupported architecture: %s\n' "$TARGETARCH" >&2; exit 1 ;; \
    esac; \
    mkdir -p /out; \
    export NIX_CONFIG="$(printf 'sandbox = false\nfilter-syscalls = false\n')"; \
    nix --extra-experimental-features 'nix-command flakes' \
        run ".#packages.$system.update-chromium" --no-update-lock-file \
        -- /out/chromium-info.json

FROM scratch AS chromium-info
COPY --from=chromium-info-build /out/chromium-info.json /chromium-info.json

FROM nix AS build
WORKDIR /src
COPY . .

ARG TARGETARCH
ARG NIX_BUILDER_IMAGE
ARG NIX_CHROMIUM=upstream
RUN --mount=type=cache,id=image-renderer-nix-${TARGETARCH}-${NIX_BUILDER_IMAGE},target=/nix,from=nix,source=/nix,sharing=locked \
    set -eu; \
    case "$TARGETARCH" in \
        arm64) system=aarch64-linux ;; \
        amd64) system=x86_64-linux ;; \
        *) printf 'Unsupported architecture: %s\n' "$TARGETARCH" >&2; exit 1 ;; \
    esac; \
    case "$NIX_CHROMIUM" in \
        upstream) image=image ;; \
        custom) image=image-custom ;; \
        *) printf 'Unsupported NIX_CHROMIUM: %s\n' "$NIX_CHROMIUM" >&2; exit 1 ;; \
    esac; \
    archive="$(nix --extra-experimental-features 'nix-command flakes' \
        build ".#packages.$system.$image" --option sandbox false \
        --option filter-syscalls false \
        --no-update-lock-file --no-link --print-out-paths)"; \
    mkdir -p /out; \
    cp "$archive" /out/image.tar.gz

FROM scratch AS archive
COPY --from=build /out/image.tar.gz /image.tar.gz
