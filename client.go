package main

// import (
// "fmt"
// "github.com/seanpont/gobro"
// "net"
// )
//
/*
Enter Address: sean@cotap.com
Enter Name: Sean Pont
Commands:
  list: list conversations
  create: <title> create and join a conversation with specified title
  say <message>: Say something in the current conversation
  leave: leave the current conversation
  exit: exit the program (and leave the current conversation)
$ list
<lists conversations>
$ join <conversation name>
<shows last 10 messages>
<messages appear as they arrive>
$ say Hi guys
Sean: Hi guys
$ leave
$
*/

// type TcpTapClient struct {
// 	address, name string
// 	db            *DB
// 	tapChan       chan Tap
// }

// func NewTcpTapClient(address, name string) *TcpTapClient {
// 	return &TcpTapClient{
// 		address: address,
// 		name:    name,
// 		db:      NewDB(address),
// 	}
// }

// func (c *TcpTapClient) Connect(service string) {
// 	conn, err := net.Dial("tcp", service)
// 	gobro.CheckErr(err)
// 	c.tapChan = connToChan(conn)
// 	go c.sync()
// 	c.run()
// }

// func (c *TcpTapClient) run() {
// 	printHelp()
// 	for {
// 		cmd, err := gobro.Prompt("$ ")
// 		if err != nil {
// 			return
// 		}
// 		switch cmd {
// 		case "help":
// 			printHelp()
// 		case "list":
// 			c.listConversations()
// 		}
// 	}
// }

// func printHelp() {
// 	print(`Available Commands:
//   list: list all conversations. Those in which you are a participant are marked with a '*'
//   create <title>: create and join a conversation with the specified title
//   join <title>: join a conversation
//   leave <title>: leave a conversation
//   open <title>: open a conversation in the window
//   close: close the current conversation
//   say <message>: Say something in the current conversation
//   exit: exit the program (and leave the current conversation)
//   help: Show this help screen
// `)
// }

// func (c *TcpTapClient) listConversations() {
// 	print("Conversations:")
// }

// func print(message string) {
// 	fmt.Print("\033[2J\033[1;1H\n\n\n\n\n\n\n\n\n\n\n\n")
// 	fmt.Print("\0337\033[1;1H", message, "\0338")
// }

// func (c *TcpTapClient) sync() {
// 	c.tapChan <- Tap{
// 		Type: "auth",
// 		User: User{
// 			Address: c.address,
// 			Name:    c.name,
// 		},
// 	}
// 	for {
// 		tap := <-c.tapChan
// 		print(fmt.Sprint(tap))
// 	}
// }
