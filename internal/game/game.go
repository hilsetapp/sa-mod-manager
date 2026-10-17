// Пакет game знает про папку GTA: San Andreas: что там лежит,
// установлен ли ModLoader и какие имена файлов спрятаны в gta3.img.
package game

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
)

// Списки стандартных имён лежат внутри программы: работают всегда.
// Рядом с exe можно положить свои lists/stock-*.txt, они добавятся к этим.
//
//go:embed lists/stock-models.txt lists/stock-maps.txt
var stockFiles embed.FS

// Info - что удалось разглядеть в папке игры.
type Info struct {
	Path         string
	Exists       bool
	LooksLikeGTA bool // есть gta_sa.exe и models/gta3.img
	HasExe       bool
	HasIMG       bool
	HasCLEO      bool
	HasModLoader bool // установлен modloader.asi
	HasASILoader bool // рядом лежит любой *.asi (значит ASI-загрузчик уже ставили)
	IMGNameCount int  // сколько имён прочитали из gta3.img
	Problems     []string

	imgNames map[string]bool // нижний регистр имён внутри архива игры
}

// Detect смотрит на папку и собирает Info. Ничего не меняет.
func Detect(dir string) Info {
	info := Info{Path: dir, imgNames: map[string]bool{}}
	if dir == "" {
		info.Problems = append(info.Problems, "папка игры не выбрана")
		return info
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		info.Problems = append(info.Problems, "такой папки нет")
		return info
	}
	info.Exists = true

	entries, err := os.ReadDir(dir)
	if err != nil {
		info.Problems = append(info.Problems, "папку не прочитать: "+err.Error())
		return info
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		switch {
		case name == "gta_sa.exe":
			info.HasExe = true
		case name == "modloader.asi":
			info.HasModLoader = true
		case strings.HasSuffix(name, ".asi"):
			info.HasASILoader = true
		case e.IsDir() && name == "cleo":
			info.HasCLEO = true
		}
	}
	img := filepath.Join(dir, "models", "gta3.img")
	if st, err := os.Stat(img); err == nil && st.Size() > 0 {
		info.HasIMG = true
		names, err := ScanIMGNames(img, 20000)
		if err != nil {
			info.Problems = append(info.Problems, "gta3.img прочитать не вышло: "+err.Error())
		} else {
			for _, n := range names {
				n = strings.ToLower(n)
				info.imgNames[n] = true
				// и имя без расширения: в списках и в модах пишут по-разному
				if ext := filepath.Ext(n); ext != "" {
					info.imgNames[strings.TrimSuffix(n, ext)] = true
				}
			}
			info.IMGNameCount = len(names)
		}
	}
	// Списки из папки lists: работают даже когда архив игры не разобрался.
	info.LoadStockLists()
	info.LooksLikeGTA = info.HasExe && info.HasIMG
	if !info.HasExe {
		info.Problems = append(info.Problems, "нет gta_sa.exe")
	}
	if !info.HasIMG {
		info.Problems = append(info.Problems, "нет models/gta3.img")
	}
	if !info.HasModLoader {
		info.Problems = append(info.Problems, "нет modloader.asi - без него включённые моды игра не увидит")
	}
	return info
}

// InStock - есть ли файл с таким именем в архиве игры.
// Если есть, значит мод не добавляет новое, а заменяет стандартный файл игры.
func (i Info) InStock(fileName string) bool {
	if i.imgNames == nil {
		return false
	}
	base := strings.ToLower(filepath.Base(fileName))
	if i.imgNames[base] {
		return true
	}
	if ext := filepath.Ext(base); ext != "" && i.imgNames[strings.TrimSuffix(base, ext)] {
		return true
	}
	return false
}

// LoadStockLists добавляет к именам архива списки стандартных файлов игры.
// Сначала встроенные, потом те, что лежат рядом с программой: так можно
// дописать свои имена, не пересобирая менеджер.
func (i *Info) LoadStockLists() int {
	if i.imgNames == nil {
		i.imgNames = map[string]bool{}
	}
	added := 0
	put := func(lines string) {
		for _, line := range strings.Split(lines, "\n") {
			word := strings.ToLower(strings.TrimSpace(line))
			if word == "" || strings.HasPrefix(word, "#") {
				continue
			}
			if !i.imgNames[word] {
				added++
			}
			i.imgNames[word] = true
		}
	}
	for _, name := range []string{"lists/stock-models.txt", "lists/stock-maps.txt"} {
		if b, err := stockFiles.ReadFile(name); err == nil {
			put(string(b))
		}
	}
	for _, dir := range listDirs() {
		for _, name := range []string{"stock-models.txt", "stock-maps.txt"} {
			if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
				put(string(b))
			}
		}
	}
	return added
}

// listDirs - где искать пользовательские списки: рядом с программой и в рабочей папке.
func listDirs() []string {
	out := []string{"."}
	if exe, err := os.Executable(); err == nil {
		out = append(out, filepath.Join(filepath.Dir(exe), "lists"))
	}
	out = append(out, "lists")
	return out
}

// StockNames - сколько имён прочитано из архива.
func (i Info) StockNames() int { return len(i.imgNames) }

// ModLoaderDir / CLEODir / ManagerDir - рабочие папки внутри игры.
func (i Info) ModLoaderDir() string { return filepath.Join(i.Path, "modloader") }
func (i Info) CLEODir() string      { return filepath.Join(i.Path, "CLEO") }
func (i Info) ManagerDir() string   { return filepath.Join(i.Path, "modmanager") }
