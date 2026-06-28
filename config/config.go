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

// Settings are app-managed UI preferences, persisted to state.toml. They live
// in their own file so the settings menu never has to rewrite the user's
// hand-maintained config.toml (which would strip comments and ordering).
type Settings struct {
	Theme           string `toml:"theme"`
	MouseWheel      bool   `toml:"mouse_wheel"`
	Animations      bool   `toml:"animations"`
	PollIntervalSec int    `toml:"poll_interval_sec"`
}

func DefaultSettings() Settings {
	return Settings{
		Theme:           "mocha",
		MouseWheel:      true,
		Animations:      true,
		PollIntervalSec: 2,
	}
}

// Dir is the config home: ~/.config/launchd-tui.
func Dir() string       { return filepath.Join(os.Getenv("HOME"), ".config", "launchd-tui") }
func ThemesDir() string { return filepath.Join(Dir(), "themes") }

func configPath() string { return filepath.Join(Dir(), "config.toml") }
func statePath() string  { return filepath.Join(Dir(), "state.toml") }
func legacyPath() string { return filepath.Join(os.Getenv("HOME"), ".launchd-tui") }

// Load reads the agent config, migrating the legacy ~/.launchd-tui dotfile into
// the config dir on first run (non-destructively — the old file is left in
// place). Returns an empty config if nothing exists yet.
func Load() (Config, error) {
	migrateLegacy()
	var cfg Config
	p := configPath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return cfg, nil
	}
	_, err := toml.DecodeFile(p, &cfg)
	return cfg, err
}

func migrateLegacy() {
	if _, err := os.Stat(configPath()); err == nil {
		return // already have a config in the new location
	}
	data, err := os.ReadFile(legacyPath())
	if err != nil {
		return // no legacy file to migrate
	}
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(configPath(), data, 0o644)
}

// LoadSettings reads state.toml, falling back to defaults for a missing file or
// any unset field.
func LoadSettings() Settings {
	s := DefaultSettings()
	if _, err := os.Stat(statePath()); err == nil {
		_, _ = toml.DecodeFile(statePath(), &s)
	}
	return s
}

func SaveSettings(s Settings) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	f, err := os.Create(statePath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(s)
}
