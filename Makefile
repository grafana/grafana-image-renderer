ARCH = darwin-x64-unknown
SKIP_CHROMIUM =
OUT =

all: clean build

clean:
	rm -rf build

deps: node_modules

node_modules: yarn.lock
	@echo "install frontend dependencies"
	yarn install --pure-lockfile --no-progress

build: clean
	./node_modules/.bin/tsc

clean_package:
	./scripts/clean_target.sh ${ARCH} ${OUT}

package:
	./scripts/package_target.sh ${ARCH} ${SKIP_CHROMIUM} ${OUT}

archive:
	./scripts/archive_target.sh ${ARCH} ${OUT}

build_package: clean clean_package build package archive
