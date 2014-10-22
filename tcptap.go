package main

import (
	"github.com/seanpont/gobro"
	"os"
)

// ===== MAIN METHOD =========================================================

func Server(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap Server <port>")

}

func Client(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap Client <host:port>")
}

func main() {
	gobro.NewCommandMap(Server, Client).Run(os.Args)
}
