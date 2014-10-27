package main

import (
	"github.com/seanpont/gobro"
	"os"
)

func main() {
	gobro.NewCommandMap(
		simpleTapServer,
		connTapServer,
		connTapClient).Run(os.Args)
}
