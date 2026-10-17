// sa-mods-cli - тот же менеджер, но без окна: удобно для скриптов и проверок.
package main

import (
	"os"

	"github.com/hilsetapp/sa-mod-manager/internal/cli"
)

var version = "dev"

func main() {
	cli.Version = version
	os.Exit(cli.Main())
}
