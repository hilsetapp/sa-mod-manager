package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// makeIMG собирает синтетический архив San Andreas: заголовок, файлы, каталог.
// dirInSectors=true - смещение каталога в блоках по 2048, как пишут настоящие инструменты.
func makeIMG(t *testing.T, names []string, dirInSectors bool) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "gta3.img")

	const sector = 2048
	dataStart := sector // первый блок отдаём под «данные»
	body := make([]byte, 0, 32*len(names))
	fileOffsets := make([]uint32, len(names))
	for i := range names {
		fileOffsets[i] = uint32(len(body) / sector)
		body = append(body, make([]byte, sector)...)
	}
	dirOffset := dataStart + len(body)
	dirBytes := make([]byte, 32*len(names))
	for i, n := range names {
		e := dirBytes[i*32 : i*32+32]
		binary.LittleEndian.PutUint32(e[0:4], fileOffsets[i])
		binary.LittleEndian.PutUint16(e[4:6], 1)
		binary.LittleEndian.PutUint16(e[6:8], 1)
		copy(e[8:32], []byte(n))
	}
	out := make([]byte, 0, dirOffset+len(dirBytes))
	out = append(out, []byte("VER2")...)
	cnt := make([]byte, 4)
	binary.LittleEndian.PutUint32(cnt, uint32(len(names)))
	out = append(out, cnt...)
	off := make([]byte, 4)
	if dirInSectors {
		binary.LittleEndian.PutUint32(off, uint32(dirOffset/sector))
	} else {
		binary.LittleEndian.PutUint32(off, uint32(dirOffset))
	}
	out = append(out, off...)
	out = append(out, make([]byte, dataStart-len(out))...)
	out = append(out, body...)
	out = append(out, dirBytes...)

	if err := os.WriteFile(p, out, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanIMGNames(t *testing.T) {
	want := []string{"infernus.dff", "infernus.txd", "handling.cfg"}
	for _, inSectors := range []bool{true, false} {
		p := makeIMG(t, want, inSectors)
		got, err := ScanIMGNames(p, 100)
		if err != nil {
			t.Fatalf("смещение в блоках=%v: %v", inSectors, err)
		}
		if len(got) != len(want) {
			t.Fatalf("получили %d имён, ждали %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("имя %d: %q, ждали %q", i, got[i], want[i])
			}
		}
	}
}

func TestScanIMGNamesRejectsOtherFormat(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "gta3.img")
	if err := os.WriteFile(p, []byte("это вообще не архив"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanIMGNames(p, 10); err == nil {
		t.Fatal("должны были вернуть ошибку на непонятном файле")
	}
}

func TestDetectEmptyFolder(t *testing.T) {
	info := Detect(t.TempDir())
	if info.LooksLikeGTA {
		t.Error("пустая папка не должна выглядеть как игра")
	}
	if len(info.Problems) == 0 {
		t.Error("про пустую папку должны быть замечания")
	}
}

func TestInStockFromLists(t *testing.T) {
	var info Info
	added := info.LoadStockLists()
	if added == 0 {
		t.Fatal("списки lists/stock-*.txt не нашлись или пустые")
	}
	if !info.InStock("models/infernus.dff") {
		t.Error("infernus.dff должен считаться стандартным файлом игры")
	}
	if !info.InStock("ambulan.dff") {
		t.Error("ambulan.dff тоже стандартный")
	}
	if info.InStock("supermod_2026.dff") {
		t.Error("выдуманный файл не должен считаться стандартным")
	}
}
