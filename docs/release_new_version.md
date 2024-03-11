# Release and publish a new version

1. Every commit to master is a possible release.
2. Version in plugin.json is used for deciding which version to release.

## Prepare

1. Update `version` and `updated` properties in plugin.json.
2. Update CHANGELOG.md.
3. Merge/push changes to master.
4. Commit is built in [Drone](https://drone.grafana.net/grafana/grafana-image-renderer).

## Promote release

1. Open [Drone](https://drone.grafana.net/grafana/grafana-image-renderer) and find the build for your commit.
2. Click on the `...` from the top-right corner to display the menu, then click on `Promote`.
3. Fill the `Create deployment` form with the values below, and click on `Deploy`:
    - `Type` = `Promote`
    - `Target` = `release` *(write it manually)*
    - *(no parameters needed)*
4. Once you've clicked on `Deploy` it will trigger a new pipeline with the release steps.

## Publish plugin to Grafana.com

Since the [migration to Drone](https://github.com/grafana/grafana-image-renderer/pull/394), this step that historically
was needed to be performed manually is no longer required and is automatically performed by `publish_to_gcom` step.

**Note:** The step will time out, but the plugin update process will continue in the background.

```
<html>
<head><title>504 Gateway Time-out</title></head>
<body>
<center><h1>504 Gateway Time-out</h1></center>
<hr><center>nginx/1.17.9</center>
</body>
</html>
```

## Deploy into Grafana Cloud
Create a PR in [Deployment Tools](https://github.com/grafana/deployment_tools/blob/master/ksonnet/lib/render-service/images.libsonnet) with the new version.
