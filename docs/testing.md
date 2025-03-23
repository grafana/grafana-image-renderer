# Testing

In order to run the image-renderer automated test suites, you need to run the following command from the root folder:

```
yarn test
```

This will launch a Grafana instance in Docker and, then, run the test suites.

_Notes:_

If there are some expected changes in the reference image files (located in `/tests/testdata`), run `yarn test-update` and push the updated references.

If the tests are failing and you want to see the difference between the image you get and the reference image, run `yarn test-diff`. This will generate images (called `diff_<test case>.png`) containing the differences in the `/tests/testdata` folder.

## Fixing Drone issues

If tests are successful in your local environement but fail in Drone. You can follow these steps to run the tests in an environment similar to the Drone pipeline. This will mount your local files of the `grafana-image-renderer` repo in the Docker image so any change that happens in the Docker image will be available in your local environment. This allows you to run `yarn test-diff` and `yarn test-update` in Docker and see the results locally. 

1. Run the Drone environment in Docker:

```
cd ./devenv/docker/drone
docker-compose up
```

2. Open a terminal within the `drone-docker-puppeteer` container and run the following commands:

```
cd /drone/src
PUPPETEER_CACHE_DIR=/drone/src/cache yarn install --frozen-lockfile --no-progress
PUPPETEER_CACHE_DIR=/drone/src/cache CI=true yarn test-ci
```

_Notes:_
The tests might take longer in the Docker container. If you run into timeout issues, you can run the test command with the `--testTimeout option`:
```
PUPPETEER_CACHE_DIR=/drone/src/cache CI=true yarn test-ci --testTimeout=10000
```
