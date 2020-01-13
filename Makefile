.PHONY: all clean deps build clean_package package archive build_package

ARCH = darwin-x64-unknown
SKIP_CHROMIUM =
OUT =
SKIP_GRPC =

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
	./scripts/package_target.sh ${ARCH} ${SKIP_CHROMIUM} ${OUT} ${SKIP_GRPC}

archive:
	./scripts/archive_target.sh ${ARCH} ${OUT}

build_package: clean clean_package build package archive
