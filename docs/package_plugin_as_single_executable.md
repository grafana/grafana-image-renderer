# Package plugin as a single executable

This plugin can be packaged into a single executable together with [Node.js](https://nodejs.org/) runtime and [Chromium](https://www.chromium.org/Home) so it doesn't require any additional dependencies to be installed.

```bash
make build_package ARCH=<arch_string>
```

Where `<arch_string>` is a combination of
- linux, darwin, win32
- ia32, x64, arm, arm64
- unknown, glibc, musl

This follows combinations allowed for GRPC plugin and you can see options [here](https://console.cloud.google.com/storage/browser/node-precompiled-binaries.grpc.io/grpc/?project=grpc-testing).

At least the following combinations have been verified to work:
- linux-x64-glibc
- darwin-x64-unknown
- win32-x64-unknown
