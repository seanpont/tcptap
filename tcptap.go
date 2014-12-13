package main

import (
	"github.com/seanpont/gobro/commander"
	"os"
)

func main() {
	commander.NewCommandMap(
		simpleTapServer,
		connTapServer,
		connTapClient).Run(os.Args)
}
