.PHONY: all clean deps build clean_package package archive build_package docker-alpine docker-debian

ARCH = darwin-x64-unknown
SKIP_CHROMIUM =
OUT =
DOCKER_TAG = dev

all: clean build

clean:
	rm -rf build

deps: node_modules

node_modules: package.json yarn.lock ## Install node modules.
	@echo "install frontend dependencies"
	yarn install --pure-lockfile --no-progress

build:
	yarn build

clean_package:
	./scripts/clean_target.sh ${ARCH} ${OUT}

package:
	./scripts/package_target.sh ${ARCH} ${SKIP_CHROMIUM} ${OUT}

archive:
	./scripts/archive_target.sh ${ARCH} ${OUT}

build_package: clean clean_package build package archive

docker-alpine:
	docker build -t grafana/grafana-image-renderer:${DOCKER_TAG} .

docker-debian:
	docker build -t grafana/grafana-image-renderer:${DOCKER_TAG}-debian -f debian.Dockerfile .

# This repository's configuration is protected (https://readme.drone.io/signature/).
# Use this make target to regenerate the configuration YAML files when
# you modify starlark files.
drone:
	drone starlark --format
	drone lint .drone.yml --trusted
	drone --server https://drone.grafana.net sign --save grafana/grafana-image-renderer
