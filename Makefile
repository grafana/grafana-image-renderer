ARCH = darwin-x64-unknown

all: clean build

clean:
	rm -rf build

build:
	./node_modules/.bin/tsc

clean_package:
	rm -rf ./plugin-${ARCH}
	rm -f ./plugin-${ARCH}.tar.gz

package:
	node scripts/pkg.js ${ARCH}
	node scripts/download_chromium.js ${ARCH}
	node scripts/download_grpc.js ${ARCH}
	cp plugin.json plugin-${ARCH}/
	cp plugin-${ARCH}/renderer plugin-${ARCH}/plugin_start_linux_amd64
	mv plugin-${ARCH}/renderer plugin-${ARCH}/plugin_start_darwin_amd64
	tar -czf plugin-${ARCH}.tar.gz plugin-${ARCH}

build_package: clean clean_package build package
