# Release and publish a new version

1. Every commit to master is a possible release.
2. Version in plugin.json is used for deciding which version to release.

## Prepare

1. Update `version` and `updated` properties in plugin.json.
2. Update CHANGELOG.md.
3. Merge/push changes to master.
4. Commit is built in [CircleCI](https://circleci.com/gh/grafana/grafana-image-renderer).

## Approve Release

1. Open [CircleCI](https://circleci.com/gh/grafana/grafana-image-renderer) and find your commit.
2. Click on `build-master` workflow link for your commit and verify build and package steps are successful (green).
4. Click on `approve-release` and approve to create GitHub release and publish docker image to Docker Hub.

## Publish plugin to Grafana.com

1. Once the GitHub release is created, run `yarn run create-gcom-plugin-json <release commit>`.

2. Push to grafana.com via `./scripts/push-to-gcom.sh` or:
```bash
JSON=$(cat ./scripts/tmp/plugin.json) gcom /plugins -X POST -H "Content-Type: application/json" -d $JSON
```

Note: The command will time-out, but the plugin update process will continue in the background.

```
<html>
<head><title>504 Gateway Time-out</title></head>
<body>
<center><h1>504 Gateway Time-out</h1></center>
<hr><center>nginx/1.17.9</center>
</body>
</html>
```

3. Inform HG team about the new release.
