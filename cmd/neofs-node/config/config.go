package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents a group of named values structured
// by tree type.
//
// Sub-trees are named configuration sub-sections,
// leaves are named configuration values.
// Names are of string type.
type Config struct {
	v *viper.Viper

	path []string
}

const (
	separator    = "."
	envPrefix    = "neofs"
	envSeparator = "_"
)

// Prm groups required parameters of the Config.
type Prm struct{}

// New creates a new Config instance.
//
// If file option is provided (WithConfigFile),
// configuration values are read from it.
// Otherwise, Config is a degenerate tree.
func New(_ Prm, opts ...Option) *Config {
	v := viper.New()

	v.SetEnvPrefix(EnvPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(separator, EnvSeparator))

	o := defaultOpts()
	for i := range opts {
		opts[i](o)
	}

	if o.path != "" {
		v.SetConfigFile(o.path)

		err := v.ReadInConfig()
		if err != nil {
			panic(fmt.Errorf("failed to read config: %v", err))
		}
	}

	return &Config{
		v: v,
	}
}
