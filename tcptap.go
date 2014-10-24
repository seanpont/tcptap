package main

import (
	"github.com/seanpont/gobro"
	"os"
)

func main() {
	gobro.NewCommandMap(
		simpleTapServer,
		streamTapServer,
		streamTapClient).Run(os.Args)
}
