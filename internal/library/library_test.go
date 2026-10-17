package library

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// makeZip собирает архив с одним общим корнем, как у модов с сайтов.
func makeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestImportZipAndToggle(t *testing.T) {
	game := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "nochka.zip")
	makeZip(t, zipPath, map[string]string{
		"Ночной город/models/infernus.dff": "модель",
		"Ночной город/night.cfg":           "настройки",
	})

	s := New(game)
	m, err := s.ImportZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Ночной город" {
		t.Errorf("название мода %q, ждали название папки внутри архива", m.Name)
	}
	if len(m.Files) != 2 {
		t.Errorf("файлов %d, ждали 2: %v", len(m.Files), m.Files)
	}

	// мод лежит выключенным
	if _, err := os.Stat(filepath.Join(game, "modmanager", "off", m.Slug)); err != nil {
		t.Fatalf("мод не в папке выключенных: %v", err)
	}

	if err := s.SetEnabled(m.Slug, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(game, "modloader", m.Slug)); err != nil {
		t.Fatalf("после включения мода нет в modloader: %v", err)
	}

	mods, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || !mods[0].Enabled {
		t.Fatalf("в списке должен быть один включённый мод: %+v", mods)
	}
	if mods[0].Summary() == "пусто" {
		t.Error("сводка по файлам пустая")
	}

	if err := s.Remove(m.Slug, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(game, "modloader", m.Slug)); err == nil {
		t.Error("папка мода осталась после удаления")
	}
}

func TestImportZipKeepsFilesInsideGame(t *testing.T) {
	game := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "bad.zip")
	makeZip(t, zipPath, map[string]string{
		"models/infernus.dff": "модель",
		"../evil.txt":         "вредное",
	})
	s := New(game)
	m, err := s.ImportZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(game, "evil.txt")); err == nil {
		t.Error("файл из архива вылез за пределы папки игры")
	}
	if len(m.Files) != 1 {
		t.Errorf("оставить надо только полезный файл, а не %v", m.Files)
	}
}

func TestUniqueSlug(t *testing.T) {
	game := t.TempDir()
	s := New(game)
	z1 := filepath.Join(t.TempDir(), "mod.zip")
	z2 := filepath.Join(t.TempDir(), "mod2.zip")
	makeZip(t, z1, map[string]string{"mod/a.dff": "x"})
	makeZip(t, z2, map[string]string{"mod/b.dff": "y"})
	m1, err := s.ImportZip(z1)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := s.ImportZip(z2)
	if err != nil {
		t.Fatal(err)
	}
	if m1.Slug == m2.Slug {
		t.Fatalf("одинаковые слаги: %q и %q", m1.Slug, m2.Slug)
	}
}

func TestWarnings(t *testing.T) {
	m := Mod{Files: []string{"gta3.img", "d3d9.dll", "readme.txt", "install.exe", "models/infernus.dff"}}
	w := m.Warnings()
	if len(w) != 3 {
		t.Fatalf("ждали три замечания, получили %d: %v", len(w), w)
	}
	if got := (Mod{Files: []string{"models/infernus.dff"}}).Warnings(); len(got) != 0 {
		t.Fatalf("обычный мод без замечаний: %v", got)
	}
}

func TestSlugifyKeepsRussianLetters(t *testing.T) {
	// слаг нужен, чтобы папку мода было видно в проводнике как есть
	cases := map[string]string{
		"Ночной город":        "ночной_город",
		"Тюнинг тачек v2":     "тюнинг_тачек_v2",
		"Пак машин (pack)":    "пак_машин_pack",
		"  ....  ":            "mod",
		"SilentPatch 1.1":     "silentpatch_1_1",
		"GTA SA / mods: best": "gta_sa_mods_best",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, ждали %q", in, got, want)
		}
	}
}
