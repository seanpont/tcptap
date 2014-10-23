package main

import (
	"github.com/seanpont/gobro"
	"os"
)

// ===== MAIN METHOD =========================================================

type Tap struct {
	Type    string `json:type`
	Sender  string `json:sender`
	Message string `json:message`
}

func Server(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap Server <port>")
	NewTcpTapServer().Serve(args[0])
}

func Client(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap Client <host:port>")
}

func main() {
	gobro.NewCommandMap(Server, Client).Run(os.Args)
}
