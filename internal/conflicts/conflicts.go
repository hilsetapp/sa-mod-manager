// Пакет conflicts ищет моды, которые мешают друг другу.
//
// Правило простое: в GTA SA файл узнаётся по имени. Если два мода кладут
// infernus.dff, победит тот, чья папка читается позже, и один из модов
// молча перестанет работать. То же самое со скриптами CLEO и плагинами ASI.
// Поэтому конфликтом мы называем одинаковый «адрес» файла у двух разных модов.
package conflicts

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/hilsetapp/sa-mod-manager/internal/library"
)

// Entry - одна группа совпадений.
type Entry struct {
	Key     string   // по чему совпали
	Kind    string   // что это за файлы
	Mods    []string // имена модов
	Enabled int      // сколько из них включено
	InStock bool     // файл с таким именем есть в архиве игры
}

// Сейчас - конфликт уже сейчас: два включённых мода делят один файл.
func (e Entry) Сейчас() bool { return e.Enabled > 1 }

// Kind - человеческое название типа файла.
func kindOf(ext string) string {
	switch strings.ToLower(ext) {
	case ".dff":
		return "модель"
	case ".txd":
		return "текстура"
	case ".col":
		return "коллизия"
	case ".cs", ".cleo":
		return "скрипт CLEO"
	case ".asi":
		return "плагин ASI"
	case ".fxt":
		return "текст CLEO"
	case ".cfg", ".ini", ".dat", ".ipl", ".ide", ".zon", ".img":
		return "файл данных"
	default:
		return "файл"
	}
}

// keyFor - по чему сравниваем файл у разных модов.
func keyFor(rel string) (key, kind string) {
	ext := strings.ToLower(filepath.Ext(rel))
	lower := strings.ToLower(filepath.ToSlash(rel))
	base := strings.ToLower(filepath.Base(rel))

	switch ext {
	case ".dff", ".txd", ".col":
		// Модели и текстуры игра ищет по имени файла, в какой бы папке он ни лежал.
		return base, kindOf(ext)
	case ".cs", ".cleo", ".asi", ".fxt":
		return base, kindOf(ext)
	default:
		// Файлы данных подменяются по относительному пути.
		trimmed := lower
		for _, p := range []string{"modloader/", "data/", "models/"} {
			trimmed = strings.TrimPrefix(trimmed, p)
		}
		return trimmed, kindOf(ext)
	}
}

// Check возвращает группы конфликтов. stock - функция «есть ли такое имя в архиве игры».
func Check(mods []library.Mod, stock func(string) bool) []Entry {
	type bucket struct {
		kind    string
		mods    []string
		enabled int
	}
	groups := map[string]*bucket{}
	for _, m := range mods {
		seenLocal := map[string]bool{} // внутри одного мода дубли не считаем
		for _, f := range m.Files {
			key, kind := keyFor(f)
			if key == "" || seenLocal[key] {
				continue
			}
			seenLocal[key] = true
			b, ok := groups[key]
			if !ok {
				b = &bucket{kind: kind}
				groups[key] = b
			}
			b.mods = append(b.mods, m.Name)
			if m.Enabled {
				b.enabled++
			}
		}
	}
	out := make([]Entry, 0, len(groups))
	for key, b := range groups {
		if len(b.mods) < 2 {
			continue
		}
		sort.Strings(b.mods)
		inStock := false
		if stock != nil {
			inStock = stock(key)
		}
		out = append(out, Entry{Key: key, Kind: b.kind, Mods: b.mods, Enabled: b.enabled, InStock: inStock})
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].Enabled > 1) != (out[j].Enabled > 1) {
			return out[i].Enabled > 1 // сначала те, что мешают прямо сейчас
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// Text - весь отчёт одной строкой, для командной строки и журнала.
func Text(entries []Entry) string {
	if len(entries) == 0 {
		return "конфликтов не нашлось"
	}
	var b strings.Builder
	for _, e := range entries {
		mark := "заметка"
		if e.Сейчас() {
			mark = "СЕЙЧАС МЕШАЮТ"
		}
		stock := ""
		if e.InStock {
			stock = ", заменяет стандартный файл"
		}
		b.WriteString("- ")
		b.WriteString(e.Key)
		b.WriteString(" (")
		b.WriteString(e.Kind)
		b.WriteString("): ")
		b.WriteString(strings.Join(e.Mods, ", "))
		b.WriteString(" [")
		b.WriteString(mark)
		b.WriteString(stock)
		b.WriteString("]\n")
	}
	return b.String()
}
