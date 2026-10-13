package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ale/metro/internal/model"
)

type Store struct {
	Dir string
}

func Default() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "metro"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "metro"), nil
}

func (s *Store) settingsPath() string { return filepath.Join(s.Dir, "settings.json") }

func (s *Store) LoadSettings() (model.Settings, error) {
	data, err := os.ReadFile(s.settingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return model.DefaultSettings(), nil
		}
		return model.Settings{}, err
	}
	var st model.Settings
	if err := json.Unmarshal(data, &st); err != nil {
		return model.Settings{}, err
	}
	st.Mode = model.NormalizeMode(st.Mode)
	st.Config.Clamp()
	return st, nil
}

func (s *Store) SaveSettings(st model.Settings) error {
	return writeJSON(s.settingsPath(), st)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
