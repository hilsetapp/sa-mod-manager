// Пакет cli - командная строка менеджера: та же библиотека, но без окна.
// Нужна для скриптов, проверок и запуска с сервера, где окна нет.
package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hilsetapp/sa-mod-manager/internal/config"
	"github.com/hilsetapp/sa-mod-manager/internal/conflicts"
	"github.com/hilsetapp/sa-mod-manager/internal/game"
	"github.com/hilsetapp/sa-mod-manager/internal/library"
)

// Version подставляет сборка (ldflags -X).
var Version = "dev"

func Main() int {
	var showVersion, doCheck, doList, doConflicts bool
	var gamePath, importZip, enable, disable string
	flag.BoolVar(&showVersion, "version", false, "печатать версию")
	flag.StringVar(&gamePath, "game", "", "папка GTA San Andreas")
	flag.BoolVar(&doCheck, "check", false, "показать, что нашлось в папке игры")
	flag.BoolVar(&doList, "list", false, "список модов")
	flag.BoolVar(&doConflicts, "conflicts", false, "проверить конфликты")
	flag.StringVar(&importZip, "import", "", "добавить мод из zip-архива")
	flag.StringVar(&enable, "enable", "", "включить мод по имени папки")
	flag.StringVar(&disable, "disable", "", "выключить мод по имени папки")
	flag.Parse()

	if showVersion {
		fmt.Printf("sa-mod-manager %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
		return 0
	}

	cfg := config.Load()
	if gamePath == "" {
		gamePath = cfg.GamePath
	}
	if gamePath != "" {
		gamePath = filepath.Clean(gamePath)
		_ = config.Save(config.Data{GamePath: gamePath})
	}

	// Командная строка: работает без окна, удобно для скриптов и проверки.
	if doCheck || doList || doConflicts || importZip != "" || enable != "" || disable != "" {
		if gamePath == "" {
			fmt.Println(`не знаю, где игра: укажи -game "C:\\Games\\GTA San Andreas"`)
			return 2
		}
		return runCLI(gamePath, doCheck, doList, doConflicts, importZip, enable, disable)
	}

	if gamePath == "" {
		// первый запуск: ищем игру в обычных местах
		if found := guessGameDir(); found != "" {
			gamePath = found
			_ = config.Save(config.Data{GamePath: gamePath})
		}
	}
	fmt.Println("это консольная версия. Окно запускается программой sa-mods.exe.")
	return 0
}

func runCLI(gamePath string, doCheck, doList, doConflicts bool, importZip, enable, disable string) int {
	info := game.Detect(gamePath)
	if doCheck {
		fmt.Println("папка:", info.Path)
		fmt.Println("похоже на GTA SA:", yesNo(info.LooksLikeGTA))
		fmt.Println("gta_sa.exe:", yesNo(info.HasExe))
		fmt.Println("models/gta3.img:", yesNo(info.HasIMG), "имена в архиве:", info.IMGNameCount)
		fmt.Println("ModLoader:", yesNo(info.HasModLoader))
		fmt.Println("папка CLEO:", yesNo(info.HasCLEO))
		for _, p := range info.Problems {
			fmt.Println("  замечание:", p)
		}
	}

	store := library.New(gamePath)
	if importZip != "" {
		m, err := store.ImportZip(importZip)
		if err != nil {
			fmt.Println("импорт не вышел:", err)
			return 1
		}
		fmt.Printf("добавлен мод %q (%s), файлов: %d\n", m.Name, m.Slug, len(m.Files))
	}
	if enable != "" || disable != "" {
		slug := enable
		on := true
		if disable != "" {
			slug, on = disable, false
		}
		if err := store.SetEnabled(slug, on); err != nil {
			fmt.Println("не вышло:", err)
			return 1
		}
		fmt.Println("ок:", slug, map[bool]string{true: "включён", false: "выключен"}[on])
	}

	mods, err := store.List()
	if err != nil {
		fmt.Println("библиотеку не прочитать:", err)
		return 1
	}
	if doList || doConflicts {
		fmt.Printf("модов в библиотеке: %d\n", len(mods))
		for _, m := range mods {
			state := "выкл"
			if m.Enabled {
				state = "вкл"
			}
			fmt.Printf("  [%s] %-28s %s\n", state, m.Name, m.Summary())
			for _, w := range m.Warnings() {
				fmt.Println("       замечание:", w)
			}
		}
	}
	if doConflicts {
		entries := conflicts.Check(mods, info.InStock)
		fmt.Println("конфликты:")
		fmt.Print(conflicts.Text(entries))
	}
	return 0
}

// guessGameDir пробует пару очевидных мест, где обычно стоит игра.
func guessGameDir() string {
	candidates := []string{
		`C:\Program Files (x86)\Rockstar Games\GTA San Andreas`,
		`C:\Program Files\Rockstar Games\GTA San Andreas`,
		`C:\Games\GTA San Andreas`,
		`D:\Games\GTA San Andreas`,
	}
	for _, c := range candidates {
		if st, err := os.Stat(filepath.Join(c, "gta_sa.exe")); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func yesNo(v bool) string {
	if v {
		return "да"
	}
	return "нет"
}
