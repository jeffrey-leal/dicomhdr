package main

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
)

//go:embed defaults/settings.json
var defaultSettingsJSON []byte

// Settings holds all persisted application preferences.
type Settings struct {
	DarkTheme      bool         `json:"darkTheme"`
	FontName       string       `json:"fontName"`
	ItalicPrivate  bool         `json:"italicPrivate"`
	MalformedColor string       `json:"malformedColor"`
	Profiles       []TagProfile `json:"profiles"`
}

func appSettingsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".dcomhdr"), nil
}

func appSettingsPath() (string, error) {
	dir, err := appSettingsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// ensureDefaultSettings creates ~/.dcomhdr/settings.json with compiled-in
// defaults if the file does not already exist.
func ensureDefaultSettings() {
	path, err := appSettingsPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return // already exists, or unwritable — not fatal
	}
	defer f.Close()
	f.Write(defaultSettingsJSON)
}

// loadSettings reads ~/.dcomhdr/settings.json.
// Compiled-in defaults are applied first, then overlaid with any saved values,
// so missing or future fields always have a sensible value.
func loadSettings() Settings {
	var s Settings
	json.Unmarshal(defaultSettingsJSON, &s) // seed with defaults

	path, err := appSettingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	json.Unmarshal(data, &s) // overlay with saved values; errors ignored
	return s
}

// saveSettings writes s to ~/.dcomhdr/settings.json as indented JSON.
func saveSettings(s Settings) {
	path, err := appSettingsPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	if s.Profiles == nil {
		s.Profiles = []TagProfile{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0o644)
}
