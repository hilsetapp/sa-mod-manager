// Пакет library ведёт библиотеку модов: папки, манифесты, включение и выключение.
//
// Включённый мод лежит в <игра>/modloader/<папка>, выключенный - в
// <игра>/modmanager/off/<папка>. Включение и выключение это перемещение папки,
// поэтому игра и моды не портятся: всё обратимо.
package library

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Mod - один мод в библиотеке.
type Mod struct {
	Slug    string    `json:"slug"`    // имя папки
	Name    string    `json:"name"`    // как показываем человеку
	Source  string    `json:"source"`  // откуда приехал (имя архива)
	Added   time.Time `json:"added"`   // когда добавлен
	Files   []string  `json:"files"`   // относительные пути файлов
	Enabled bool      `json:"enabled"` // сейчас в игре
}

// Types - сколько чего в моде, по расширениям.
func (m Mod) Types() map[string]int {
	out := map[string]int{}
	for _, f := range m.Files {
		ext := strings.ToLower(filepath.Ext(f))
		if ext == "" {
			ext = "без расширения"
		}
		out[ext]++
	}
	return out
}

// Summary - короткая строка «что внутри».
func (m Mod) Summary() string {
	t := m.Types()
	order := []string{".dff", ".txd", ".cs", ".asi", ".col", ".fxt", ".ini", ".cfg"}
	parts := make([]string, 0, 6)
	seen := map[string]bool{}
	for _, ext := range order {
		if n := t[ext]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s: %d", strings.TrimPrefix(ext, "."), n))
			seen[ext] = true
		}
	}
	rest := 0
	for ext, n := range t {
		if !seen[ext] {
			rest += n
		}
	}
	if rest > 0 {
		parts = append(parts, fmt.Sprintf("прочее: %d", rest))
	}
	if len(parts) == 0 {
		return "пусто"
	}
	return strings.Join(parts, ", ")
}

// Warnings - на что посмотреть в моде до включения.
// Это не запрет, а подсказка: такие файлы чаще всего и ломают сборку.
func (m Mod) Warnings() []string {
	out := []string{}
	seen := map[string]bool{}
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, f := range m.Files {
		base := strings.ToLower(filepath.Base(f))
		ext := strings.ToLower(filepath.Ext(base))
		switch base {
		case "gta3.img", "gta_int.img":
			add("подменяет весь архив игры (" + base + "): после него конфликты почти гарантированы")
		case "d3d9.dll":
			add("кладёт d3d9.dll - обычно это графика вроде ENB или ReShade, просто смотри, откуда мод")
		case "dinput8.dll":
			add("кладёт dinput8.dll - это бывает и загрузчик ASI, и старый хук")
		}
		if ext == ".exe" || ext == ".bat" || ext == ".cmd" {
			add("внутри есть " + base + ": моды так не ставятся, это мусор в архиве")
		}
	}
	return out
}

// Store - библиотека модов одной игры.
type Store struct {
	gameDir string
}

// New открывает библиотеку для папки игры.
func New(gameDir string) *Store { return &Store{gameDir: gameDir} }

func (s *Store) managerDir() string { return filepath.Join(s.gameDir, "modmanager") }
func (s *Store) offDir() string     { return filepath.Join(s.managerDir(), "off") }
func (s *Store) metaDir() string    { return filepath.Join(s.managerDir(), "meta") }
func (s *Store) onDir() string      { return filepath.Join(s.gameDir, "modloader") }

// Ensure создаёт рабочие папки, если их нет.
func (s *Store) Ensure() error {
	for _, d := range []string{s.offDir(), s.metaDir(), s.onDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// List собирает все моды: из папки выключенных и из modloader.
func (s *Store) List() ([]Mod, error) {
	if err := s.Ensure(); err != nil {
		return nil, err
	}
	mods := []Mod{}
	seen := map[string]bool{}
	// выключенные
	entries, err := os.ReadDir(s.offDir())
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := s.load(e.Name(), false)
		if err != nil {
			continue
		}
		mods = append(mods, m)
		seen[e.Name()] = true
	}
	// включённые
	entries, err = os.ReadDir(s.onDir())
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || seen[e.Name()] {
			continue
		}
		m, err := s.load(e.Name(), true)
		if err != nil {
			continue
		}
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool { return strings.ToLower(mods[i].Name) < strings.ToLower(mods[j].Name) })
	return mods, nil
}

func (s *Store) metaPath(slug string) string { return filepath.Join(s.metaDir(), slug+".json") }

func (s *Store) load(slug string, enabled bool) (Mod, error) {
	dir := s.dir(slug, enabled)
	m := Mod{Slug: slug, Name: slug, Enabled: enabled}
	if b, err := os.ReadFile(s.metaPath(slug)); err == nil {
		_ = json.Unmarshal(b, &m)
		m.Slug = slug
		m.Enabled = enabled
	}
	files, err := walkFiles(dir)
	if err != nil {
		return m, err
	}
	m.Files = files
	return m, nil
}

func (s *Store) dir(slug string, enabled bool) string {
	if enabled {
		return filepath.Join(s.onDir(), slug)
	}
	return filepath.Join(s.offDir(), slug)
}

func (s *Store) save(m Mod) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.metaDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(m.Slug), b, 0o644)
}

// walkFiles собирает относительные пути файлов внутри папки.
func walkFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}

// SetEnabled включает или выключает мод перемещением папки.
func (s *Store) SetEnabled(slug string, on bool) error {
	from := s.dir(slug, !on)
	to := s.dir(slug, on)
	if _, err := os.Stat(from); err != nil {
		// уже в нужном состоянии
		if _, err2 := os.Stat(to); err2 == nil {
			return nil
		}
		return fmt.Errorf("папки мода %q нет", slug)
	}
	if _, err := os.Stat(to); err == nil {
		return fmt.Errorf("папка %q уже занята", filepath.Base(to))
	}
	if err := os.Rename(from, to); err != nil {
		return err
	}
	return nil
}

// Remove удаляет мод вместе с его файлами и манифестом.
func (s *Store) Remove(slug string, enabled bool) error {
	if err := os.RemoveAll(s.dir(slug, enabled)); err != nil {
		return err
	}
	_ = os.Remove(s.metaPath(slug))
	return nil
}

// ImportZip разбирает zip-архив и кладёт мод в библиотеку (выключенным).
// Внутри архива находим корень: если там одна папка, берём её содержимое.
func (s *Store) ImportZip(zipPath string) (Mod, error) {
	if err := s.Ensure(); err != nil {
		return Mod{}, err
	}
	name := strings.TrimSuffix(filepath.Base(zipPath), filepath.Ext(zipPath))

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return Mod{}, errors.New("это не zip-архив: " + err.Error())
	}
	defer r.Close()

	// Если внутри архива одна общая папка, берём её имя как название мода:
	// люди обычно называют архив криво, а папку нормально.
	prefix := commonRoot(r.File)
	if root := strings.TrimSuffix(prefix, "/"); root != "" {
		name = root
	}
	slug := uniqueSlug(s.offDir(), slugify(name))
	dst := filepath.Join(s.offDir(), slug)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return Mod{}, err
	}
	for _, f := range r.File {
		rel := strings.TrimPrefix(f.Name, prefix)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" || strings.HasSuffix(rel, "/") {
			continue
		}
		out := filepath.Join(dst, filepath.FromSlash(rel))
		if !strings.HasPrefix(out, dst) {
			continue // защита от «..» в архиве
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return Mod{}, err
		}
		in, err := f.Open()
		if err != nil {
			return Mod{}, err
		}
		outF, err := os.Create(out)
		if err != nil {
			in.Close()
			return Mod{}, err
		}
		_, copyErr := io.Copy(outF, in)
		in.Close()
		outF.Close()
		if copyErr != nil {
			return Mod{}, copyErr
		}
	}
	files, err := walkFiles(dst)
	if err != nil {
		return Mod{}, err
	}
	if len(files) == 0 {
		_ = os.RemoveAll(dst)
		return Mod{}, errors.New("в архиве не нашлось файлов")
	}
	m := Mod{Slug: slug, Name: name, Source: filepath.Base(zipPath), Added: time.Now(), Files: files}
	if err := s.save(m); err != nil {
		return Mod{}, err
	}
	return m, nil
}

// commonRoot ищет единственную общую папку у файлов архива.
func commonRoot(files []*zip.File) string {
	root := ""
	for _, f := range files {
		name := f.Name
		if strings.HasSuffix(name, "/") {
			continue
		}
		parts := strings.Split(name, "/")
		if len(parts) < 2 {
			return ""
		}
		if root == "" {
			root = parts[0] + "/"
			continue
		}
		if !strings.HasPrefix(name, root) {
			return ""
		}
	}
	return root
}

// slugify делает из названия имя папки: буквы и цифры оставляем,
// пробелы и разделители превращаем в «_». Русские буквы не переводим
// в латиницу: так папку всегда видно такой, как назвал её автор мода.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var out []rune
	lastUnderscore := false
	for _, r := range s {
		switch {
		case r < 128 && ((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')):
			out = append(out, r)
			lastUnderscore = false
		case r >= 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			out = append(out, r)
			lastUnderscore = false
		case r == ' ' || r == '_' || r == '-' || r == '.':
			if !lastUnderscore && len(out) > 0 {
				out = append(out, '_')
				lastUnderscore = true
			}
		}
	}
	res := strings.Trim(string(out), "_")
	if len([]rune(res)) > 48 {
		res = string([]rune(res)[:48])
		res = strings.Trim(res, "_")
	}
	if res == "" {
		res = "mod"
	}
	return res
}

func uniqueSlug(dir, slug string) string {
	cand := slug
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, cand)); err != nil {
			return cand
		}
		cand = fmt.Sprintf("%s_%d", slug, i)
	}
}
