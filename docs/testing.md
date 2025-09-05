# Testing

In order to run the image-renderer automated test suites, you need to run the following command from the root folder:

```
yarn test
```

This will launch a Grafana instance in Docker and, then, run the test suites.

_Notes:_

If there are some expected changes in the reference image files (located in `/tests/testdata`), run `yarn test-update` and push the updated references.

If the tests are failing and you want to see the difference between the image you get and the reference image, run `yarn test-diff`. This will generate images (called `diff_<test case>.png`) containing the differences in the `/tests/testdata` folder.
