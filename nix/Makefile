NIX_SYSTEM ?= $(shell uname -m | sed 's/arm64/aarch64/')-linux
NIX_IMAGE ?= image-renderer-nix

.PHONY: image-nix
image-nix: | $(OUT_DIR)
	case "$(NIX_SYSTEM)" in \
		aarch64-linux) platform=linux/arm64 ;; \
		x86_64-linux) platform=linux/amd64 ;; \
		*) printf 'Unsupported NIX_SYSTEM: %s\n' "$(NIX_SYSTEM)" >&2; exit 1 ;; \
	esac; \
	docker build --platform "$$platform" --file nix/Dockerfile.nix \
		--output "type=local,dest=$(OUT_DIR)/nix/$(NIX_SYSTEM)" .
	docker load --input "$(OUT_DIR)/nix/$(NIX_SYSTEM)/image.tar.gz"
	docker tag grafana-image-renderer:nix "$(NIX_IMAGE)"

.PHONY: update-nix-chromium
update-nix-chromium:
	docker build --file nix/Dockerfile.nix --target chromium-info --no-cache-filter chromium-info-build \
		--output type=local,dest=nix .
