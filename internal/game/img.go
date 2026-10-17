package game

import (
	"encoding/binary"
	"errors"
	"os"
	"strings"
)

// Архив San Andreas (models/gta3.img) устроен так:
//
//	0..3   - подпись «VER2»
//	4..7   - сколько файлов внутри
//	8..11  - где начинается каталог
//
// Дальше идут записи по 32 байта: смещение, размеры и имя файла в 24 байтах.
// Разные сборки и инструменты пишут смещение то в байтах, то в блоках по 2048,
// поэтому мы пробуем несколько вариантов и берём тот, где каталог выглядит
// как список имён файлов. Читаем только имена, ничего не меняем.
func ScanIMGNames(path string, limit int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	head := make([]byte, 12)
	if _, err := f.ReadAt(head, 0); err != nil {
		return nil, err
	}
	if string(head[0:4]) != "VER2" {
		return nil, errors.New("архив не в формате San Andreas (нет подписи VER2)")
	}
	count := binary.LittleEndian.Uint32(head[4:8])
	if count == 0 || count > 1_000_000 {
		return nil, errors.New("странное число файлов в архиве")
	}
	raw := binary.LittleEndian.Uint32(head[8:12])

	candidates := []int64{
		int64(raw) * 2048, // так пишет большинство инструментов
		int64(raw),        // редкий вариант: смещение в байтах
		12,                // редкий вариант: каталог сразу после заголовка
	}
	max := int(count)
	if limit > 0 && max > limit {
		max = limit
	}
	buf := make([]byte, 32*max)

	for _, off := range candidates {
		if off < 0 || off+int64(len(buf)) > st.Size() {
			continue
		}
		if _, err := f.ReadAt(buf, off); err != nil {
			continue
		}
		names, ok := parseDir(buf, max)
		if ok {
			return names, nil
		}
	}
	return nil, errors.New("каталог архива не разобрался: формат незнакомый")
}

// parseDir вытаскивает имена из записей каталога и проверяет, что это похоже на правду.
func parseDir(buf []byte, count int) ([]string, bool) {
	names := make([]string, 0, count)
	good := 0
	for i := 0; i < count; i++ {
		entry := buf[i*32 : i*32+32]
		raw := entry[8:32]
		end := 0
		for end < len(raw) && raw[end] != 0 {
			end++
		}
		if end == 0 {
			continue
		}
		name := string(raw[:end])
		if !looksLikeName(name) {
			return nil, false
		}
		names = append(names, name)
		good++
	}
	// Похоже на каталог, если имена нашлись ну хоть где-то рядом с ожидаемым количеством.
	if good == 0 {
		return nil, false
	}
	return names, true
}

// looksLikeName - имя файла игры: только печатные символы, точка, длина до 23.
func looksLikeName(s string) bool {
	if len(s) == 0 || len(s) > 23 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-' || r == ' ':
		default:
			return false
		}
	}
	return strings.Contains(s, ".")
}
