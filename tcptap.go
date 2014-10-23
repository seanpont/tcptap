package main

import (
	"encoding/json"
	"github.com/seanpont/gobro"
	"net"
	"os"
)

type Tap struct {
	Type         string       `json:type`
	Message      string       `json:message`
	User         User         `json:user`
	Conversation Conversation `json:conversation`
}

func connToChan(conn net.Conn) chan Tap {
	tapChan := make(chan Tap)

	// Outbox
	go func(conn net.Conn, tapChan chan Tap) {
		encoder := json.NewEncoder(conn)
		for {
			tap := <-tapChan
			err := encoder.Encode(tap)
			if err != nil {
				gobro.LogErr(err)
				return
			}
		}
	}(conn, tapChan)

	// Inbox
	go func(conn net.Conn, tapChan chan Tap) {
		decoder := json.NewDecoder(conn)
		for {
			var tap Tap
			err := decoder.Decode(&tap)
			if err != nil {
				gobro.LogErr(err)
				return
			}
			tapChan <- tap
		}
	}(conn, tapChan)

	return tapChan
}

// ===== MAIN METHOD =========================================================

func Server(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap Server <port>")
	NewTcpTapServer().Serve(args[0])
}

func Client(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap Client <host:port>")
	// address, _ := gobro.Prompt("Address: ")
	// name, _ := gobro.Prompt("Name: ")
	address, name := "sean@cotap.com", "Sean Pont"
	NewTcpTapClient(address, name).Connect(args[0])
}

func main() {
	gobro.NewCommandMap(
		SimpleTap).Run(os.Args)
}
