package config

import (
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"
)

func FromConfig(path string, envVars ...string) cli.ValueSourceChain {
	var sources []cli.ValueSource
	for _, envVar := range envVars {
		sources = append(sources, cli.EnvVar(envVar))
	}
	sources = append(sources,
		yaml.YAML(path, altsrc.StringSourcer("config.yaml")),
		yaml.YAML(path, altsrc.StringSourcer("config.yml")),
		// All JSON is valid YAML, so we can just... load JSON as YAML!
		yaml.YAML(path, altsrc.StringSourcer("config.json")))

	return cli.NewValueSourceChain(sources...)
}
