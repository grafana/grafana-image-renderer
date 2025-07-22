# Release and publish a new version

Remember that every commit to master is a possible release.

## Prepare

1. Update `version` and `updated` properties in plugin.json.
2. Update CHANGELOG.md.
3. Merge/push changes to master.

## Release

Tag a new version, either in the GitHub UI or pushing it from the CLI:

```
$ git switch master
$ git fetch
$ git reset --hard origin/master
$ git tag v0.0.0
$ git push origin v0.0.0
```

GitHub Actions takes care of _everything_ until you get to the Grafana Cloud deployment.

## Deploy into Grafana Cloud

Create a PR in [Deployment Tools](https://github.com/grafana/deployment_tools/blob/master/ksonnet/lib/render-service/images.libsonnet) with the new version.
