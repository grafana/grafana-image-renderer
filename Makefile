ARCH = darwin-x64-unknown

all: clean build

clean:
	rm -rf build

build:
	./node_modules/.bin/tsc

clean_package:
	rm -rf ./plugin-${ARCH}
	rm -f ./plugin-${ARCH}.zip

package:
	node scripts/pkg.js ${ARCH}
	node scripts/download_chromium.js ${ARCH}
	node scripts/download_grpc.js ${ARCH}
	node scripts/rename_executable.js ${ARCH}
	cp plugin.json plugin-${ARCH}/
	zip -yqr plugin-${ARCH}.zip plugin-${ARCH}

build_package: clean clean_package build package
