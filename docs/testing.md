# Testing

In order to run the image-renderer automated test suites, you need to run the following command from the root folder:

```
yarn test
```

This will launch a Grafana instance in Docker and, then, run the test suites.

_Notes:_

If there are some expected changes in the reference image files (located in `/tests/testdata`), run `yarn test-update` and push the updated references.

If the tests are failing and you want to see the difference between the image you get and the reference image, run `yarn test-diff`. This will generate images (called `diff_<test case>.png`) containing the differences in the `/tests/testdata` folder.

## Test docker image

The docker image has tests that you can run on it.

`IMAGE=X go test ./tests/acceptance/...`

If you [built the Docker image from source](./building_from_source.md#docker-image) you can test it with

`IMAGE=custom-grafana-image-renderer go test ./tests/acceptance/...`

Or you can also pull a specific image to test, for example:

`docker image pull grafana/grafana-image-renderer:v4.0.13`

And then run the tests

`IMAGE=grafana/grafana-image-renderer:v4.0.13 go test ./tests/acceptance/...`

### Enterprise tests

Some tests require an active Enterprise licence.

If you're a Grafana Labs employee, you can find one of these in the `grafana-enterprise` repository:

```shell
$ ln -s ../grafana-enterprise/tools/license.jwt .
```
