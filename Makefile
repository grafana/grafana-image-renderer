all: clean build

clean:
	rm -rf dist

build:
	tsc

package:
	rm -rf ./plugin
	rm ./plugin.zip
	./node_modules/.bin/pkg -t node10 . --out-path plugin
	mkdir -p plugin/puppeteer
	cp -R ./node_modules/puppeteer/.local-chromium plugin/puppeteer
	cp node_modules/grpc/src/node/extension_binary/node-v64-darwin-x64-unknown/grpc_node.node plugin/
	cp plugin.json plugin/
	mv plugin/renderer plugin/plugin_start_darwin_amd64
	zip -rq plugin plugin
