## 2.1.0 (2021-05-11)

- Chore/Security: Upgrade dependencies and bump Node to LTS (14.16.1) [#218](https://github.com/grafana/grafana-image-renderer/pull/218), [AgnesToulet](https://github.com/AgnesToulet)

## 2.0.1 (2021-01-26)

- Browser: Use timeout parameter for initial navigation to the dashboard being rendered [#171](https://github.com/grafana/grafana-image-renderer/pull/171), [](https://github.com/w4rgrum)

## 2.0.0 (2020-05-16)

- Plugin: Migrate to @grpc/grpc-js to resolve problems when IPv6 is disabled [#135](https://github.com/grafana/grafana-image-renderer/pull/135), [aknuds1](https://github.com/aknuds1)
- Adds support for new Grafana backend plugin system [#128](https://github.com/grafana/grafana-image-renderer/pull/128), [marefr](https://github.com/marefr)
- Browser: Adds support for setting viewport device scale factor [#128](https://github.com/grafana/grafana-image-renderer/pull/128), [marefr](https://github.com/marefr)
- Browser: Adds support for attaching Accept-Language header to support render is name locale as Grafana user [#128](https://github.com/grafana/grafana-image-renderer/pull/128), [marefr](https://github.com/marefr)
- Browser: Fail render if the URL has socket protocol [#127](https://github.com/grafana/grafana-image-renderer/pull/127), [aknuds1](https://github.com/aknuds1)
- Chore: Upgrade typescript dependencies [#129](https://github.com/grafana/grafana-image-renderer/pull/129), [marefr](https://github.com/marefr)

## 2.0.0-beta1 (2020-04-22)

- Adds support for new Grafana backend plugin system [#128](https://github.com/grafana/grafana-image-renderer/pull/128), [marefr](https://github.com/marefr)
- Browser: Adds support for setting viewport device scale factor [#128](https://github.com/grafana/grafana-image-renderer/pull/128), [marefr](https://github.com/marefr)
- Browser: Adds support for attaching Accept-Language header to support render is name locale as Grafana user [#128](https://github.com/grafana/grafana-image-renderer/pull/128), [marefr](https://github.com/marefr)
- Browser: Fail render if the URL has socket protocol [#127](https://github.com/grafana/grafana-image-renderer/pull/127), [aknuds1](https://github.com/aknuds1)
- Chore: Upgrade typescript dependencies [#129](https://github.com/grafana/grafana-image-renderer/pull/129), [marefr](https://github.com/marefr)

## 1.0.12 (2020-03-31)

- Remote rendering: Delete temporary file after serving it to client [#120](https://github.com/grafana/grafana-image-renderer/pull/120), [marefr](https://github.com/marefr)
- Remote rendering: More configuration options [#123](https://github.com/grafana/grafana-image-renderer/pull/123), [marefr](https://github.com/marefr)

## 1.0.12-beta1 (2020-03-30)

- Remote rendering: More configuration options [#123](https://github.com/grafana/grafana-image-renderer/pull/123), [marefr](https://github.com/marefr)

## 1.0.11 (2020-03-20)

- Render: Add support for enabling verbose logging using environment variable [#105](https://github.com/grafana/grafana-image-renderer/pull/105), [marefr](https://github.com/marefr)
- Render: Fix panel titles should not be focused when rendering [#114](https://github.com/grafana/grafana-image-renderer/pull/114), [AgnesToulet](https://github.com/AgnesToulet)
- Security: Upgrade minimist dependency to v1.2.5 [#118](https://github.com/grafana/grafana-image-renderer/pull/118), [marefr](https://github.com/marefr)

## 1.0.10 (2020-02-18)

- Plugin: Fix unable to start Grafana (Windows) with version 1.0.8 and 1.0.9 [#103](https://github.com/grafana/grafana-image-renderer/pull/103), [marefr](https://github.com/marefr)

## 1.0.9 (2020-01-30)

- Remote rendering: Improve error handling, logging and metrics [#92](https://github.com/grafana/grafana-image-renderer/pull/92), [marefr](https://github.com/marefr)
  - Service: Don't swallow exceptions and fix logging of parameters
  - Metrics: Use status 499 when client close the connection
  - Docker: Set NODE_ENV=production
  - Changed request logging to use debug level if status < 400 and error if >= 400
- Plugin: Adds icon [#95](https://github.com/grafana/grafana-image-renderer/pull/95), [marefr](https://github.com/marefr)

## 1.0.8 (2020-01-20)

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
