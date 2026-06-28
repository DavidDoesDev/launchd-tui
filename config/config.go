package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Agent struct {
	Label string `toml:"label"`
	Name  string `toml:"name"`
}

func (a Agent) DisplayName() string {
	if a.Name != "" {
		return a.Name
	}
	return a.Label
}

type Config struct {
	Agents []Agent `toml:"agents"`
}

func Load() (Config, error) {
	path := filepath.Join(os.Getenv("HOME"), ".launchd-tui")
	var cfg Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	_, err := toml.DecodeFile(path, &cfg)
	return cfg, err
}
