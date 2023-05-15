## 3.7.1 (2023-05-15)
- Docker: remove alpine edge repo [#413](https://github.com/grafana/grafana-image-renderer/pull/413), [sozercan](https://github.com/sozercan)
- Bump yaml from 2.1.1 to 2.2.2 [#421](https://github.com/grafana/grafana-image-renderer/pull/421), [dependabot[bot]](https://github.com/apps/dependabot)

## 3.7.0 (2023-04-17)
- Security: can set array of auth tokens [#417](https://github.com/grafana/grafana-image-renderer/pull/417), [AgnesToulet](https://github.com/AgnesToulet)
- Bump pkg from 5.8.0 to 5.8.1 [#415](https://github.com/grafana/grafana-image-renderer/pull/415), [AgnesToulet](https://github.com/AgnesToulet)
- Bump jimp from 0.16.1 to 0.16.13 [#406](https://github.com/grafana/grafana-image-renderer/pull/406), [AgnesToulet](https://github.com/AgnesToulet)

## 3.6.4 (2023-02-10)

- Add Snyk workflow [#402](https://github.com/grafana/grafana-image-renderer/pull/402), [SadFaceSmith](https://github.com/SadFaceSmith)
- Fix null error [#403](https://github.com/grafana/grafana-image-renderer/pull/403), [spinillos](https://github.com/spinillos)

## 3.6.3 (2023-01-11)

- Add support for page zooming option [#387](https://github.com/grafana/grafana-image-renderer/pull/387), [kaffarell](https://github.com/kaffarell)
- Migrate from CircleCI to Drone [#394](https://github.com/grafana/grafana-image-renderer/pull/394), [spinillos](https://github.com/spinillos), [joanlopez](https://github.com/joanlopez)

## 3.6.2 (2022-10-22)

- Log errors related to JSHandle@object as debug [#376](https://github.com/grafana/grafana-image-renderer/pull/376), [spinillos](https://github.com/spinillos)
- Chore: Update Puppeteer deprecated functions [#375](https://github.com/grafana/grafana-image-renderer/pull/375), [spinillos](https://github.com/spinillos)
- Fix: Update _client with _client() to avoid to fail when creating a CSV [#372](https://github.com/grafana/grafana-image-renderer/pull/372), [spinillos](https://github.com/spinillos)
- Chore: Update all dependencies [#369](https://github.com/grafana/grafana-image-renderer/pull/369), [DanCech](https://github.com/DanCech)

## 3.6.1 (2022-08-30)

- Chore: Update to Node 16 [#365](https://github.com/grafana/grafana-image-renderer/pull/365), [Clarity-89](https://github.com/Clarity-89)
- Update waiting condition for full page screenshots [#362](https://github.com/grafana/grafana-image-renderer/pull/362), [spinillos](https://github.com/spinillos)
- Fix invalid Content-Disposition [#357](https://github.com/grafana/grafana-image-renderer/pull/357), [spinillos](https://github.com/spinillos)

## 3.6.0 (2022-08-16)

- Security: Add support for auth token [#364](https://github.com/grafana/grafana-image-renderer/pull/364), [xlson](https://github.com/xlson), [joanlopez](https://github.com/joanlopez)

## 3.5.0 (2022-07-18)

- Added File Sanitization API with [DOMPurify](https://github.com/cure53/DOMPurify) as the backend. [#349](https://github.com/grafana/grafana-image-renderer/pull/349), [ArturWierzbicki](https://github.com/ArturWierzbicki)
- Security: upgrade dependencies [#356](https://github.com/grafana/grafana-image-renderer/pull/356), [#348](https://github.com/grafana/grafana-image-renderer/pull/348), [#347](https://github.com/grafana/grafana-image-renderer/pull/347),  [AgnesToulet](https://github.com/AgnesToulet)

## 3.4.2 (2022-03-23)

- Security: upgrade dependencies [#337](https://github.com/grafana/grafana-image-renderer/pull/337), [AgnesToulet](https://github.com/AgnesToulet)
- Fix: set captureBeyondViewport to false by default to fix rendering old panels [#335](https://github.com/grafana/grafana-image-renderer/pull/335), [AgnesToulet](https://github.com/AgnesToulet)

## 3.4.1 (2022-02-23)

- Fix: replace `sharp` with `jimp` to resolve issues with installing native dependencies [#325](https://github.com/grafana/grafana-image-renderer/pull/325), [ArturWierzbicki](https://github.com/ArturWierzbicki)

## 3.4.0 (2022-02-17)

- Support new concurrency mode: contextPerRenderKey [#314](https://github.com/grafana/grafana-image-renderer/pull/314), [ArturWierzbicki](https://github.com/ArturWierzbicki)
- Support full height dashboards and scaled thumbnails [#312](https://github.com/grafana/grafana-image-renderer/pull/312), [ryantxu](https://github.com/ryantxu)

## 3.3.0 (2021-11-18)

- Chore: Bump pkg from 5.3.3 to 5.4.1 [#305](https://github.com/grafana/grafana-image-renderer/pull/305), [AgnesToulet](https://github.com/AgnesToulet)
- Configuration: Add timeout setting for clustered mode [#303](https://github.com/grafana/grafana-image-renderer/pull/303), [AgnesToulet](https://github.com/AgnesToulet)

## 3.2.1 (2021-10-07)

- Chore: Upgrade dev dependencies [#294](https://github.com/grafana/grafana-image-renderer/pull/294), [AgnesToulet](https://github.com/AgnesToulet)
- Chore: Fix eslint usage [#293](https://github.com/grafana/grafana-image-renderer/pull/293), [AgnesToulet](https://github.com/AgnesToulet)
- Docs: Fix links in README.md [#290](https://github.com/grafana/grafana-image-renderer/pull/290), [simonc6372](https://github.com/simonc6372)
- Security: Bump semver-regex from 3.1.2 to 3.1.3 [#289](https://github.com/grafana/grafana-image-renderer/pull/289), [dependabot[bot]](https://github.com/apps/dependabot)

## 3.2.0 (2021-09-17)

- Docs: Update documentation to improve visibility and avoid duplicates with Grafana documentation [#277](https://github.com/grafana/grafana-image-renderer/pull/277), [AgnesToulet](https://github.com/AgnesToulet)
- Instrumentation: Update grafana_image_renderer_step_duration_seconds buckets [#287](https://github.com/grafana/grafana-image-renderer/pull/287), [AgnesToulet](https://github.com/AgnesToulet)
- Security: Bump chokidar from 3.5.1 to 3.5.2 [#284](https://github.com/grafana/grafana-image-renderer/pull/284), [AgnesToulet](https://github.com/AgnesToulet)
- Instrumentation: Add gauge of total number of requests in flight [#281](https://github.com/grafana/grafana-image-renderer/pull/281), [AgnesToulet](https://github.com/AgnesToulet)
- Security: Bump axios from 0.21.1 to 0.21.4 [#283](https://github.com/grafana/grafana-image-renderer/pull/283), [dependabot[bot]](https://github.com/apps/dependabot)
- Chore: Add self-contained setup for load test [#275](https://github.com/grafana/grafana-image-renderer/pull/275), [pianohacker](https://github.com/pianohacker)

## 3.1.0 (2021-09-01)

- Settings: Set the maximum device scale factor to 4 as default [#276](https://github.com/grafana/grafana-image-renderer/pull/276), [AgnesToulet](https://github.com/AgnesToulet)
- Metrics: Add browser timing metrics [#263](https://github.com/grafana/grafana-image-renderer/pull/263), [AgnesToulet](https://github.com/AgnesToulet)
- Settings: Add --disable-gpu in the default Chromium args [#262](https://github.com/grafana/grafana-image-renderer/pull/262), [AgnesToulet](https://github.com/AgnesToulet)
- Security: Update path-parse to v1.0.7 [#268](https://github.com/grafana/grafana-image-renderer/pull/268), [joanlopez](https://github.com/joanlopez)
- Chore: Upgrade dependencies [#246](https://github.com/grafana/grafana-image-renderer/pull/246), [Clarity-89](https://github.com/Clarity-89)
- Docker: Run image renderer under non-root Grafana user [#144](https://github.com/grafana/grafana-image-renderer/pull/144), [wardbekker](https://github.com/wardbekker)

### Important change

The default Chromium flags have been updated to include `--disable-gpu` as it fixes memory leaks issues when using the `default` rendering mode. If you don't want to use this flag, you need to update your service configuration, either through the [service configuration file](https://github.com/grafana/grafana-image-renderer/blob/master/docs/remote_rendering_using_docker.md#configuration-file), the [environment variables](https://github.com/grafana/grafana-image-renderer/blob/master/docs/remote_rendering_using_docker.md#environment-variables) or the [Grafana configuration file](https://grafana.com/docs/grafana/latest/administration/configuration/#rendering_args) (if you're using the plugin mode).

## 3.0.1 (2021-06-10)

- Browser: Fix panel timezone when the timezone query parameter is specified [#224](https://github.com/grafana/grafana-image-renderer/pull/224), [Bujupah](https://github.com/Bujupah)
- Docker: Fix version endpoint for Docker images [#248](https://github.com/grafana/grafana-image-renderer/pull/248), [mbentley](https://github.com/mbentley)

## 3.0.0 (2021-06-07)

- Security: Bump path-parse from 1.0.6 to 1.0.7 [#244](https://github.com/grafana/grafana-image-renderer/pull/244), [AgnesToulet](https://github.com/AgnesToulet)
- HTTP Server: Add version endpoint to get the current version [#239](https://github.com/grafana/grafana-image-renderer/pull/239), [AgnesToulet](https://github.com/AgnesToulet)
- Security: Bump ws from 7.4.5 to 7.4.6 [#238](https://github.com/grafana/grafana-image-renderer/pull/238), [dependabot[bot]](https://github.com/apps/dependabot)
- Remove support for plugin V1 protocol [#233](https://github.com/grafana/grafana-image-renderer/pull/233), [AgnesToulet](https://github.com/AgnesToulet)
- Browser: Fix moving CSV file when the tmp folder is not on the same device as the target file path [#232](https://github.com/grafana/grafana-image-renderer/pull/232), [AgnesToulet](https://github.com/AgnesToulet)
- Chore: Upgrade grabpl version [#231](https://github.com/grafana/grafana-image-renderer/pull/231), [AgnesToulet](https://github.com/AgnesToulet)
- Add CSV rendering feature [#217](https://github.com/grafana/grafana-image-renderer/pull/217), [AgnesToulet](https://github.com/AgnesToulet)

## 3.0.0-beta2 (2021-05-26)

- Remove support for plugin V1 protocol [#233](https://github.com/grafana/grafana-image-renderer/pull/233), [AgnesToulet](https://github.com/AgnesToulet)
- Browser: Fix moving CSV file when the tmp folder is not on the same device as the target file path [#232](https://github.com/grafana/grafana-image-renderer/pull/232), [AgnesToulet](https://github.com/AgnesToulet)
- Chore: Upgrade grabpl version [#231](https://github.com/grafana/grafana-image-renderer/pull/231), [AgnesToulet](https://github.com/AgnesToulet)

## 3.0.0-beta1 (2021-05-19)

- Add CSV rendering feature [#217](https://github.com/grafana/grafana-image-renderer/pull/217), [AgnesToulet](https://github.com/AgnesToulet)

## 2.1.1 (2021-05-18)

- Chore: Add changelog in package files [#226](https://github.com/grafana/grafana-image-renderer/pull/226), [AgnesToulet](https://github.com/AgnesToulet)

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
