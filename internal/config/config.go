package config

import (
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const EnvPrefix = "BOTNET_"

func NewConfig() (*koanf.Koanf, error) {
	conf := koanf.New(".")

	err := conf.Load(file.Provider("config.toml"), toml.Parser())
	if err != nil {
		return nil, err
	}

	err = conf.Load(env.Provider(".", env.Opt{
		Prefix: EnvPrefix,
		TransformFunc: func(k, v string) (string, any) {
			return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(k, EnvPrefix), "_", ".")), v
		},
	}), nil)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
