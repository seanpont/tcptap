package main

import (
	"encoding/json"
	"fmt"
	"github.com/seanpont/gobro"
	"net"
	"sync"
)

// ===== MODEL ===============================================================

type Message struct {
	User string
	Body string
}

type Conversation struct {
	Title    string
	Users    []string
	Messages []Message
}

// ===== TAP PROTOCOL ========================================================

type Tap struct {
	Type   string            `json:"type"`
	Value  string            `json:"value"`
	Params map[string]string `json:"params"`
}

// Types
const ERROR = "error"
const AUTH = "auth"
const CREATE = "create"

// Params
const CLOSE = "close"

// ===== NETWORKING ==========================================================

func connToChan(conn net.Conn) (inbox chan Tap, outbox chan Tap) {
	inbox = make(chan Tap)
	outbox = make(chan Tap)

	// Outbox
	go func() {
		encoder := json.NewEncoder(conn)
		for {
			tap := <-outbox
			err := encoder.Encode(tap)
			if err != nil || tap.Params[CLOSE] != "" {
				fmt.Println("Outbox closed")
				conn.Close()
				return
			}
		}
	}()

	// Inbox
	go func() {
		decoder := json.NewDecoder(conn)
		for {
			var tap Tap
			err := decoder.Decode(&tap)
			if err != nil {
				fmt.Println("Inbox closed")
				return
			}
			inbox <- tap
		}
	}()

	return
}

// ===== SERVER ==============================================================

func streamTapServer(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap streamTapServer <port>")
	NewStreamTapServer().listen(args[0])
}

type StreamTapServer struct {
	sync.Mutex
	taps          []Tap
	conversations []Conversation
	tapChans      map[string]chan Tap
}

func NewStreamTapServer() *StreamTapServer {
	return &StreamTapServer{
		taps:          make([]Tap, 0),
		conversations: make([]Conversation, 0),
		tapChans:      make(map[string]chan Tap),
	}
}

func (s *StreamTapServer) listen(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	gobro.CheckErr(err)
	fmt.Println("StreamTapServer listening on port", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			gobro.LogErr(err)
			continue
		}
		inbox, outbox := connToChan(conn)
		go s.handleConn(inbox, outbox)
	}
}

func (s *StreamTapServer) handleConn(inbox chan Tap, outbox chan Tap) {

	// First tap must be of type 'auth'
	authTap := <-inbox
	if authTap.Type != AUTH || authTap.Value == "" {
		outbox <- Tap{
			Type:   ERROR,
			Value:  "First tap must be of Type 'auth' with non-empty value",
			Params: map[string]string{CLOSE: "true"},
		}
		return
	}

	// User is verified. Send them all relevant taps.
	user := authTap.Value
	s.Lock()
	for _, tap := range s.taps {
		if s.isRelevant(tap, user) {
			outbox <- tap
		}
	}

	// Add them to the list of tapchans
	s.tapChans[user] = outbox
	s.Unlock()

	// listen for taps
	for {
		s.handleTap(<-inbox)
	}
}

func (s *StreamTapServer) isRelevant(tap Tap, user string) bool {
	return true
}

func (s *StreamTapServer) handleTap(tap Tap) {
	fmt.Println("handleTap:", tap)
}

// ===== Client ===============================================================

func streamTapClient(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap streamTapClient <host:port>")
	NewStreamTapClient().connect(args[0])
}

type StreamTapClient struct {
}

func NewStreamTapClient() *StreamTapClient {
	return &StreamTapClient{}
}

func (c *StreamTapClient) connect(service string) {

	printHelp()
	gobro.Prompt("Name: ")
}

func printHelp() {
	print(`Available Commands:
  list: list all conversations. Those in which you are a participant are marked with a '*'
  create <title>: create and join a conversation with the specified title
  join <title>: join a conversation
  leave <title>: leave a conversation
  open <title>: open a conversation in the window
  close: close the current conversation
  say <message>: Say something in the current conversation
  exit: exit the program (and leave the current conversation)
  help: Show this help screen
`)
}

func print(message string) {
	fmt.Print("\033[2J\033[1;1H\n\n\n\n\n\n\n\n\n\n\n\n")
	fmt.Print("\0337\033[1;1H", message, "\0338")
}
