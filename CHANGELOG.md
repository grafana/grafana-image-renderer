## 1.0.8 (2019-01-20)

- Build: Upgrade Node.js requirement to LTS (v12) [#57](https://github.com/grafana/grafana-image-renderer/pull/57), [marefr](https://github.com/marefr)
- Docker: Add unifont font to support rendering other language, like Chinese/Japanese [#75](https://github.com/grafana/grafana-image-renderer/pull/75), [okhowang](https://github.com/okhowang)
- Subscribing to page events to catch errors from browser [#88](https://github.com/grafana/grafana-image-renderer/pull/88), [marefr](https://github.com/marefr)
- Plugin: Automatically assign grpc port per default [#87](https://github.com/grafana/grafana-image-renderer/pull/87), [marefr](https://github.com/marefr)
- Plugin: Support configuring default timezone thru environment variable [#86](https://github.com/grafana/grafana-image-renderer/pull/86), [marefr](https://github.com/marefr)
- Remote rendering: Support configuring default timezone thru config file and environment variable [#86](https://github.com/grafana/grafana-image-renderer/pull/86), [marefr](https://github.com/marefr)
- Remote rendering: Support configuring HTTP host and port thru config and environment variables [#40](https://github.com/grafana/grafana-image-renderer/pull/40), [marefr](https://github.com/marefr)
- Remote rendering: Support reading config from file [#73](https://github.com/grafana/grafana-image-renderer/pull/73), [marefr](https://github.com/marefr)
- Remote rendering: Collect and expose Prometheus metrics [#71](https://github.com/grafana/grafana-image-renderer/pull/71), [marefr](https://github.com/marefr)
### Breaking changes
- Plugin now automatically assigns gPRC port not in use. Before port `50059` was used. You can change this by using the `GF_RENDERER_PLUGIN_GRPC_PORT` environment variable.

## 1.0.8-beta1 (2019-12-17)

- Remote rendering: Collect and expose Prometheus metrics [#71](https://github.com/grafana/grafana-image-renderer/pull/71), [marefr](https://github.com/marefr)
- Build: Upgrade Node.js requirement to LTS (v12) [#57](https://github.com/grafana/grafana-image-renderer/pull/57), [marefr](https://github.com/marefr)

## 1.0.7 (2019-12-03)

- Provide correctly named config parameter to Chromium when overriding to skip https errors using environment variable `GF_RENDERER_PLUGIN_IGNORE_HTTPS_ERRORS` and/or `IGNORE_HTTPS_ERRORS` [#62](https://github.com/grafana/grafana-image-renderer/pull/62), [marefr](https://github.com/marefr)

## 1.0.6 (2019-11-25)

- Wait until all network connections to be idle before rendering [#24](https://github.com/grafana/grafana-image-renderer/pull/24), [d1ff](https://github.com/d1ff)
- Support ignoring https errors using environment variable [#59](https://github.com/grafana/grafana-image-renderer/pull/59), [marefr](https://github.com/marefr)
- Docker: Update dependencies to remove vulnerabilities [#53](https://github.com/grafana/grafana-image-renderer/pull/53), [marefr](https://github.com/marefr)
- Fix typo in log statement [#39](https://github.com/grafana/grafana-image-renderer/pull/39), [ankon](https://github.com/ankon)
- Updated documentation

## 1.0.5 (2019-09-11)

- Include md5 checksums in release artifacts

## 1.0.4 (2019-09-11)

- Update readme and docs

## 1.0.3 (2019-09-10)

- Automate docker release

## 1.0.2 (2019-09-10)

- Don't include dist directory in archive (zip) files

## 1.0.1 (2019-09-09)

- Switch docker base image from node:10 to node:alpine-10 [#36](https://github.com/grafana/grafana-image-renderer/issues/36), [marefr](https://github.com/marefr)
- Updated the panel render wait function to account for Grafana version 6 [#26](https://github.com/grafana/grafana-image-renderer/issues/26), [bmichaelis](https://github.com/bmichaelis)
- Updated dependencies

## 1.0.0 (2019-08-16)

Initial release containing prebuilt binaries available for download. Right now the binaries themselves should be considered alpha as they need more testing.
