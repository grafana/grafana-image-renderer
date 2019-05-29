all: clean build

clean:
	rm -rf dist

build:
	tsc

package:
	rm -rf ./plugin
	rm -f ./plugin.tar.gz
	./node_modules/.bin/pkg -t node10 . --out-path plugin
	node download_chromium.js mac
	node download_grpc.js darwin-x64-unknown
	cp plugin.json plugin/
	mv plugin/renderer plugin/plugin_start_darwin_amd64
	tar -czf plugin.tar.gz plugin
