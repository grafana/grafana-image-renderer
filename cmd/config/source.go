package config

import (
	"fmt"
	"slices"

	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"
)

func FromConfig(path string, additionally ...cli.ValueSource) cli.ValueSourceChain {
	return cli.NewValueSourceChain(
		slices.Concat(additionally, []cli.ValueSource{
			yaml.YAML(path, altsrc.StringSourcer("config.yaml")),
			yaml.YAML(path, altsrc.StringSourcer("config.yml")),
			// All JSON is valid YAML, so we can just... load JSON as YAML!
			yaml.YAML(path, altsrc.StringSourcer("config.json")),
		})...,
	)
}

var FromEnv = cli.EnvVar

func Mapping(source cli.ValueSource, mapper func(string) string) cli.ValueSource {
	return &mappedValueSource{
		underlying: source,
		mapper:     mapper,
	}
}

var (
	_ cli.ValueSource    = (*mappedValueSource)(nil)
	_ cli.EnvValueSource = (*mappedValueSource)(nil)
)

type mappedValueSource struct {
	underlying cli.ValueSource
	mapper     func(string) string
}

func (m *mappedValueSource) String() string {
	return m.underlying.String()
}

func (m *mappedValueSource) GoString() string {
	mapper := any(m.mapper) // otherwise we fail compilation with: fmt.Sprintf format %#v arg m.mapper is a func value, not called
	return fmt.Sprintf("&mappedValueSource{underlying:%#v, mapper:%#v}", m.underlying, mapper)
}

func (m *mappedValueSource) Lookup() (string, bool) {
	if v, ok := m.underlying.Lookup(); ok {
		return m.mapper(v), true
	}
	return "", false
}

func (m *mappedValueSource) IsFromEnv() bool {
	if evs, ok := m.underlying.(cli.EnvValueSource); ok {
		return evs.IsFromEnv()
	}
	return false
}

func (m *mappedValueSource) Key() string {
	if evs, ok := m.underlying.(cli.EnvValueSource); ok {
		return evs.Key()
	}
	return ""
}
