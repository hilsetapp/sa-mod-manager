// Пакет profiles хранит наборы сборок: какие моды включены в этом профиле.
package profiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Profile struct {
	Name  string    `json:"name"`
	Mods  []string  `json:"mods"` // слаги включённых модов
	Saved time.Time `json:"saved"`
}

type Store struct{ dir string }

func New(gameDir string) *Store {
	return &Store{dir: filepath.Join(gameDir, "modmanager", "profiles")}
}

func (s *Store) List() ([]Profile, error) {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := []Profile{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var p Profile
		if json.Unmarshal(b, &p) != nil {
			continue
		}
		if p.Name == "" {
			p.Name = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) Save(p Profile) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	if p.Saved.IsZero() {
		p.Saved = time.Now()
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r > 127:
			return r
		case r == ' ' || r == '-' || r == '_':
			return '_'
		default:
			return -1
		}
	}, p.Name)
	if name == "" {
		name = "profile"
	}
	return os.WriteFile(filepath.Join(s.dir, name+".json"), b, 0o644)
}

func (s *Store) Delete(name string) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasSuffix(strings.TrimSuffix(e.Name(), ".json"), name) {
			if err := os.Remove(filepath.Join(s.dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
