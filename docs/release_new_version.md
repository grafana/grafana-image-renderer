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

1. Download `md5sums.txt` from the GitHub release of the version you want to publish.

### Alternative 1 - Open a PR in grafana-plugin-repository

1. Checkout [grafana-plugin-repository](https://github.com/grafana/grafana-plugin-repository)
2. Update repo.json with new plugin version JSON payload
3. Commit, push and open a PR

See [PR](https://github.com/grafana/grafana-plugin-repository/pull/479) for an example.

### Alternative 2 - Push directly to grafana.com

plugin_version.json:

```json
{
  "url": "https://github.com/grafana/grafana-image-renderer",
  "commit": "a32a8a1f538cf5138319616704dd769f8cf7c116",
  "download": {
    "darwin-amd64": {
      "url": "https://github.com/grafana/grafana-image-renderer/releases/download/v1.0.1/plugin-darwin-x64-unknown.zip",
      "md5": "329d4d5020f8e626d3661d1aae21d810"
    },
    "linux-amd64": {
      "url": "https://github.com/grafana/grafana-image-renderer/releases/download/v1.0.1/plugin-linux-x64-glibc.zip",
      "md5": "6f36cffad5b55e339ba29ae1ec369abf"
    },
    "windows-amd64": {
      "url": "https://github.com/grafana/grafana-image-renderer/releases/download/v1.0.1/plugin-win32-x64-unknown.zip",
      "md5": "d1f7c0a407eeb198d82358a94d884778"
    }
  }
}
```

```bash
JSON=$(cat plugin_version.json) gcom /plugins -X POST -H "Content-Type: application/json" -d $JSON
```
