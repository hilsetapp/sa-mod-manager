package conflicts

import (
	"testing"

	"github.com/hilsetapp/sa-mod-manager/internal/library"
)

func TestSameModelInTwoMods(t *testing.T) {
	mods := []library.Mod{
		{Name: "Тюнинг тачек", Enabled: true, Files: []string{"infernus.dff", "infernus.txd"}},
		{Name: "Пак машин", Enabled: true, Files: []string{"models/infernus.dff"}},
	}
	got := Check(mods, nil)
	if len(got) != 1 {
		t.Fatalf("ждали один конфликт, получили %d: %v", len(got), got)
	}
	if got[0].Key != "infernus.dff" || !got[0].Сейчас() {
		t.Errorf("не то: %+v", got[0])
	}
}

func TestDifferentTexturesDoNotConflict(t *testing.T) {
	mods := []library.Mod{
		{Name: "A", Files: []string{"infernus.dff"}},
		{Name: "B", Files: []string{"sultan.dff"}},
	}
	if got := Check(mods, nil); len(got) != 0 {
		t.Fatalf("конфликтов быть не должно: %v", got)
	}
}

func TestDisabledPairIsOnlyWarning(t *testing.T) {
	mods := []library.Mod{
		{Name: "A", Files: []string{"script.cs"}},
		{Name: "B", Files: []string{"script.cs"}},
	}
	got := Check(mods, nil)
	if len(got) != 1 || got[0].Сейчас() {
		t.Fatalf("два выключенных мода это заметка, а не конфликт: %v", got)
	}
}

func TestDataFilesCompareByPath(t *testing.T) {
	mods := []library.Mod{
		{Name: "A", Files: []string{"data/handling.cfg"}},
		{Name: "B", Files: []string{"handling.cfg"}},
	}
	got := Check(mods, nil)
	if len(got) != 1 {
		t.Fatalf("файлы данных должны совпасть по пути: %v", got)
	}
	if got[0].Kind != "файл данных" {
		t.Errorf("тип не тот: %s", got[0].Kind)
	}
}

func TestStockMark(t *testing.T) {
	mods := []library.Mod{
		{Name: "A", Files: []string{"infernus.dff"}},
		{Name: "B", Files: []string{"infernus.dff"}},
	}
	got := Check(mods, func(name string) bool { return name == "infernus.dff" })
	if len(got) != 1 || !got[0].InStock {
		t.Fatalf("должны были отметить, что файл есть в архиве игры: %v", got)
	}
}
