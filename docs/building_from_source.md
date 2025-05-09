# Build the image renderer from source

Git clone this repo:

```bash
git clone https://github.com/grafana/grafana-image-renderer.git
cd grafana-image-renderer
```

## Plugin

1. Install [Node.js](https://nodejs.org/) and [Yarn](https://yarnpkg.com/en/).
2. Install dependencies and build:

    ```bash
    make deps

    # build and package for Linux x64
    make build_package ARCH=linux-x64-glibc

    # build and package for Window x64
    make build_package ARCH=win32-x64-unknown

    # build and package for Darwin x64
    make build_package ARCH=darwin-x64-unknown

    # build and package for Mac ARM64
    make build_package ARCH=darwin-arm64-unknown

    # build and package without including Chromium
    make build_package ARCH=<ARCH> SKIP_CHROMIUM=true OUT=plugin-<ARCH>-no-chromium
    ```

3. Built artifacts can be found in ./artifacts directory

## Docker image

1. Install Docker
2. Build Docker image:

    ```bash
    docker build -t custom-grafana-image-renderer .
    ```

## Local Node.js application using local Chrome/Chromium

1. Install [Node.js](https://nodejs.org/) and [Yarn](https://yarnpkg.com/en/).
2. Install dependencies and build:

    ```bash
    make deps
    make build
    ```

3. Built artifacts are found in ./build