// Пакет config помнит последнюю папку игры между запусками.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Data struct {
	GamePath string `json:"game_path"`
}

func file() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "sa-mod-manager", "config.json")
}

func Load() Data {
	var d Data
	b, err := os.ReadFile(file())
	if err == nil {
		_ = json.Unmarshal(b, &d)
	}
	return d
}

func Save(d Data) error {
	p := file()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
