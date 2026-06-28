package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/DavidDoesDev/launchd-tui/config"
	"github.com/charmbracelet/lipgloss"
)

// Theme is the set of named color roles the UI draws from. Built-in themes are
// compiled in; users add their own as TOML files under the config dir's
// themes/ directory (see config.LoadThemes), which are parsed into this struct.
type Theme struct {
	Name     string         `toml:"name"`
	Base     lipgloss.Color `toml:"base"`
	Mantle   lipgloss.Color `toml:"mantle"`
	Surface0 lipgloss.Color `toml:"surface0"`
	Surface1 lipgloss.Color `toml:"surface1"`
	Overlay0 lipgloss.Color `toml:"overlay0"`
	Subtext0 lipgloss.Color `toml:"subtext0"`
	Text     lipgloss.Color `toml:"text"`
	Green    lipgloss.Color `toml:"green"`
	GreenDim lipgloss.Color `toml:"green_dim"`
	Red      lipgloss.Color `toml:"red"`
	Yellow   lipgloss.Color `toml:"yellow"`
	Blue     lipgloss.Color `toml:"blue"`
	Mauve    lipgloss.Color `toml:"mauve"`
}

// BuiltinThemes are shipped in the binary, in display order.
var BuiltinThemes = []Theme{mochaTheme, latteTheme, gruvboxTheme, nordTheme}

// AllThemes returns the built-in themes followed by any user themes found as
// TOML files under the config dir's themes/ directory. A user theme whose name
// matches a built-in overrides it.
func AllThemes() []Theme {
	themes := append([]Theme(nil), BuiltinThemes...)
	for _, ut := range loadUserThemes(config.ThemesDir()) {
		replaced := false
		for i := range themes {
			if themes[i].Name == ut.Name {
				themes[i] = ut
				replaced = true
				break
			}
		}
		if !replaced {
			themes = append(themes, ut)
		}
	}
	return themes
}

// themeByName looks up a theme by name across built-ins + user themes.
func themeByName(name string) (Theme, bool) {
	for _, t := range AllThemes() {
		if t.Name == name {
			return t, true
		}
	}
	return Theme{}, false
}

func loadUserThemes(dir string) []Theme {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Theme
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		var t Theme
		if _, err := toml.DecodeFile(filepath.Join(dir, e.Name()), &t); err != nil {
			continue
		}
		if t.Name == "" {
			t.Name = strings.TrimSuffix(e.Name(), ".toml")
		}
		out = append(out, t)
	}
	return out
}

var mochaTheme = Theme{
	Name: "mocha",
	Base: "#1e1e2e", Mantle: "#181825", Surface0: "#313244", Surface1: "#45475a",
	Overlay0: "#6c7086", Subtext0: "#a6adc8", Text: "#cdd6f4",
	Green: "#a6e3a1", GreenDim: "#4a6741", Red: "#f38ba8", Yellow: "#f9e2af",
	Blue: "#89b4fa", Mauve: "#cba6f7",
}

var latteTheme = Theme{
	Name: "latte",
	Base: "#eff1f5", Mantle: "#e6e9ef", Surface0: "#ccd0da", Surface1: "#bcc0cc",
	Overlay0: "#9ca0b0", Subtext0: "#6c6f85", Text: "#4c4f69",
	Green: "#40a02b", GreenDim: "#8cb87a", Red: "#d20f39", Yellow: "#df8e1d",
	Blue: "#1e66f5", Mauve: "#8839ef",
}

var gruvboxTheme = Theme{
	Name: "gruvbox",
	Base: "#282828", Mantle: "#1d2021", Surface0: "#3c3836", Surface1: "#504945",
	Overlay0: "#928374", Subtext0: "#bdae93", Text: "#ebdbb2",
	Green: "#b8bb26", GreenDim: "#5f6f1e", Red: "#fb4934", Yellow: "#fabd2f",
	Blue: "#83a598", Mauve: "#d3869b",
}

var nordTheme = Theme{
	Name: "nord",
	Base: "#2e3440", Mantle: "#292e39", Surface0: "#3b4252", Surface1: "#434c5e",
	Overlay0: "#4c566a", Subtext0: "#d8dee9", Text: "#eceff4",
	Green: "#a3be8c", GreenDim: "#5e7150", Red: "#bf616a", Yellow: "#ebcb8b",
	Blue: "#81a1c1", Mauve: "#b48ead",
}
