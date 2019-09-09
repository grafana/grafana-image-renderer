ARCH = darwin-x64-unknown

all: clean build

clean:
	rm -rf build

deps: node_modules

node_modules: package.json yarn.lock
	@echo "install frontend dependencies"
	yarn install --pure-lockfile --no-progress

build:
	./node_modules/.bin/tsc

clean_package:
	rm -rf .dist/plugin-${ARCH}
	rm -f ./dist/artifacts/plugin-${ARCH}.zip

package:
	./scripts/package_target.sh ${ARCH}

archive:
	./scripts/archive_target.sh ${ARCH}

build_package: clean clean_package build package
