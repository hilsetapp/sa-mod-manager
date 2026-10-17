// sa-mods - окно менеджера модов GTA San Andreas.
// Здесь только точка входа: вся работа лежит в internal.
package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/hilsetapp/sa-mod-manager/internal/config"
	"github.com/hilsetapp/sa-mod-manager/internal/gui"
)

var version = "dev"

func main() {
	var gamePath string
	flag.StringVar(&gamePath, "game", "", `папка GTA San Andreas, например "C:\Games\GTA San Andreas"`)
	flag.Parse()

	if gamePath == "" {
		gamePath = config.Load().GamePath
	} else {
		gamePath = filepath.Clean(gamePath)
		_ = config.Save(config.Data{GamePath: gamePath})
	}
	gui.Run(gamePath, version)
	os.Exit(0)
}
