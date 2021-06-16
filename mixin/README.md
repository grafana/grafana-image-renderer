# Grafana Render service Mixin

To build, you need to have `mixtool` and `jsonnetfmt` installed. If you
have a working Go development environment, it's easiest to run the following:

```bash
$ go get github.com/monitoring-mixins/mixtool/cmd/mixtool
$ go get github.com/google/go-jsonnet/cmd/jsonnetfmt
```

You can then build the Prometheus rules files `alerts.yaml` and
`rules.yaml` and a directory `dashboard_out` with the JSON dashboard files
for Grafana render service:

```bash
$ make build
```

For more advanced uses of mixins, see
https://github.com/monitoring-mixins/docs.
