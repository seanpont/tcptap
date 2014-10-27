package main

import (
	"encoding/json"
	"fmt"
	"github.com/seanpont/gobro"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
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

type Data struct {
	Taps          []*Tap
	Users         map[string]bool
	Conversations map[string]*Conversation
}

func NewData() *Data {
	return &Data{
		Taps:          make([]*Tap, 0),
		Users:         make(map[string]bool),
		Conversations: make(map[string]*Conversation),
	}
}

func (d *Data) Update(tap *Tap) {
	switch tap.Type {
	case TYPE_AUTH:
		if tap.User != "" {
			d.Users[tap.User] = true
		}
	case TYPE_CONVERSATION:
		fmt.Println("Create conversation", tap.Conversation, d.Conversations[tap.Conversation])
		if tap.Conversation != "" && d.Conversations[tap.Conversation] == nil {
			d.Conversations[tap.Conversation] = NewConversation(tap)
		}
	case TYPE_MESSAGE:
		if tap.Conversation != "" && tap.Value != "" {
			conversation := d.Conversations[tap.Conversation]
			if conversation != nil {
				conversation.SendMessage(tap)
			}
		}
	case TYPE_INVITE:
		if tap.Conversation != "" && tap.Value != "" {
			user := tap.Value
			conversation := d.Conversations[tap.Conversation]
			if conversation != nil {
				d.Users[user] = true
				conversation.AddUser(user)
			}
		}
	}
}

func NewConversation(tap *Tap) *Conversation {
	c := Conversation{
		Title:    tap.Conversation,
		Users:    make([]string, 0),
		Messages: make([]Message, 0),
	}
	c.Users = append(c.Users, tap.User)
	for user, _ := range tap.Params {
		c.Users = append(c.Users, user)
	}
	if tap.Value != "" {
		c.SendMessage(tap)
	}
	return &c
}

func (c *Conversation) SendMessage(tap *Tap) {
	c.Messages = append(c.Messages, Message{
		User: tap.User,
		Body: tap.Value,
	})
}

func (c *Conversation) AddUser(user string) {
	if !gobro.Contains(c.Users, user) {
		c.Users = append(c.Users, user)
	}
}

// ===== TAP PROTOCOL ========================================================

type Tap struct {
	Id           int               `json:"id"`
	Type         string            `json:"type"`
	User         string            `json:"user"`
	Conversation string            `json:"conversation"`
	Value        string            `json:"value"`
	Params       map[string]string `json:"params"`
}

func NewTap(_type, user, conversation, value string, params ...string) *Tap {
	tap := Tap{
		Type:         _type,
		User:         user,
		Conversation: conversation,
		Value:        value,
	}
	if len(params) > 0 {
		tap.Params = make(map[string]string)
		for i := 0; i < len(params); i += 2 {
			tap.Params[params[i]] = params[i+1]
		}
	}
	return &tap
}

const (
	// Types
	TYPE_ERROR        = "error"
	TYPE_AUTH         = "auth"
	TYPE_CONVERSATION = "conversation"
	TYPE_MESSAGE      = "message"
	TYPE_INVITE       = "invite"
	PARAM_TAP_ID      = "tapId"
)

// ===== NETWORKING ==========================================================

func connToChan(conn net.Conn) (<-chan *Tap, chan<- *Tap) {
	inbox := make(chan *Tap)
	outbox := make(chan *Tap)

	// Outbox
	go func() {
		defer conn.Close()
		encoder := json.NewEncoder(conn)
		for {
			tap, ok := <-outbox
			if !ok {
				fmt.Println("Outbox closed")
				return
			}
			err := encoder.Encode(tap)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error encoding tap:", tap, err)
			}
		}
	}()

	// Inbox
	go func() {
		decoder := json.NewDecoder(conn)
		for {
			tap := new(Tap)
			err := decoder.Decode(tap)
			if err != nil {
				if err != io.EOF {
					fmt.Println("Error decoding tap:", err)
				}
				close(inbox)
				return
			}
			inbox <- tap
		}
	}()

	return inbox, outbox
}

// ===== SERVER ==============================================================

type ConnTapServer struct {
	data        *Data
	tapChans    map[string]chan bool
	tapChanLock sync.Mutex
	tapCore     chan *Tap
}

func connTapServer(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap streamTapServer <port>")
	NewConnTapServer().listen(args[0])
}

func NewConnTapServer() *ConnTapServer {
	s := &ConnTapServer{
		data:     NewData(),
		tapChans: make(map[string]chan bool),
		tapCore:  make(chan *Tap, 100),
	}
	go s.processTaps()
	return s
}

func (s *ConnTapServer) processTaps() {
	for {
		tap := <-s.tapCore
		fmt.Println("Processing: ", tap)
		s.data.Update(tap)

		tap.Id = len(s.data.Taps)
		s.data.Taps = append(s.data.Taps, tap)

		for user, tapChan := range s.tapChans {
			if s.isRelevant(user, tap) {
				notify(tapChan)
			}
		}
	}
}

func (s *ConnTapServer) listen(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	gobro.CheckErr(err)
	fmt.Println("ConnTapServer listening on port", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			gobro.LogErr(err)
			continue
		}
		inbox, outbox := connToChan(conn)
		go s.handle(inbox, outbox)
	}
}

func (s *ConnTapServer) handle(inbox <-chan *Tap, outbox chan<- *Tap) {
	defer close(outbox)

	// The first tap must be an auth tap
	authTap, ok := <-inbox
	if !ok {
		return
	}
	if authTap.Type != TYPE_AUTH || authTap.User == "" {
		outbox <- &Tap{
			Type:  TYPE_ERROR,
			Value: "First tap must be auth with valid user",
		}
		return
	}

	user := authTap.User
	tapCursor := 0
	tapIdStr := authTap.Params[PARAM_TAP_ID]
	if tapIdStr != "" {
		tapCursor, _ = strconv.Atoi(tapIdStr)
	}

	tapChan := make(chan bool, 1)

	// register tapChan
	s.tapChanLock.Lock()
	oldTapChan := s.tapChans[user]
	s.tapChans[user] = tapChan // in with the new
	s.tapChanLock.Unlock()

	// if another user was connected, kill them
	if oldTapChan != nil {
		oldTapChan <- false // out with the old
	}

	// deregister when we're done
	defer func() {
		s.tapChanLock.Lock()
		if s.tapChans[user] == tapChan {
			delete(s.tapChans, user)
		}
		s.tapChanLock.Unlock()
	}()

	s.tapCore <- authTap
	notify(tapChan) // prime the pump - effectively the 'catch up' tap
	for {
		select {
		case tap, ok := <-inbox:
			if !ok {
				return
			}
			tap.User = user
			s.tapCore <- tap
		case alive, ok := <-tapChan:
			if !alive || !ok {
				return
			}
			// advance tap cursor
			for ; tapCursor < len(s.data.Taps); tapCursor++ {
				tap := s.data.Taps[tapCursor]
				if s.isRelevant(user, tap) {
					outbox <- tap
				}
			}
		}
	}
}

func (s *ConnTapServer) isRelevant(user string, tap *Tap) bool {
	return true
}

func notify(tapChan chan<- bool) {
	select {
	case tapChan <- true:
	default:
	}
}

// ===== Client ===============================================================

func connTapClient(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap connTapClient <host:port>")
	name, _ := gobro.Prompt("Please enter your name: ")
	NewConnTapClient(name).connect(args[0])
}

type ConnTapClient struct {
	user         string
	data         *Data
	conversation string
}

func NewConnTapClient(user string) *ConnTapClient {
	return &ConnTapClient{
		user: user,
		data: NewData(),
	}
}

func (c *ConnTapClient) connect(service string) {
	fmt.Println("Connecting...")
	conn, err := net.Dial("tcp", service)
	gobro.CheckErr(err)
	defer conn.Close()
	fmt.Println("Connected!")

	inbox, outbox := connToChan(conn)
	userToSync := make(chan *Tap)
	syncToUser := make(chan *Tap)
	go c.sync(inbox, outbox, userToSync, syncToUser)
	c.handle(syncToUser, userToSync)
}

func (c *ConnTapClient) sync(
	inbox <-chan *Tap, outbox chan<- *Tap, userToSync <-chan *Tap, syncToUser chan<- *Tap) {

	defer close(outbox)

	// Authentication
	outbox <- NewTap(TYPE_AUTH, c.user, "", "")

	// Listen loop
	for {
		select {
		case tap, ok := <-inbox:
			if !ok {
				return
			}
			c.data.Update(tap)
			syncToUser <- tap
		case tap, ok := <-userToSync:
			if !ok {
				fmt.Println("userToSync closed")
				return
			}
			outbox <- tap
		}
	}
}

func (c *ConnTapClient) handle(syncToUser <-chan *Tap, userToSync chan<- *Tap) {
	defer func() {
		fmt.Println("handle: close userToSync")
		close(userToSync)
	}()

	prompt := make(chan string)
	go func() {
		defer close(prompt)
		for {
			cmd, err := gobro.Prompt("")
			if err != nil {
				return
			}
			prompt <- cmd
		}
	}()

	for {
		select {
		case tap := <-syncToUser:
			print(fmt.Sprint(tap))
		case cmd, ok := <-prompt:
			if !ok {
				print("Goodbye")
				return
			}
			c.handleCmd(cmd, userToSync)
		}
	}
}

func (c *ConnTapClient) handleCmd(message string, userToSync chan<- *Tap) {
	parts := strings.Split(message, " ")
	cmd := parts[0]
	switch cmd {
	case "help":
		printHelp()
	case "list":
		s := make([]string, 0)
		for title, _ := range c.data.Conversations {
			s = append(s, title)
		}
		if len(s) == 0 {
			print("No conversations found")
		} else {
			print(strings.Join(s, "\n"))
		}
	case "create":
		title := parts[1]
		userToSync <- &Tap{
			Type:         TYPE_CONVERSATION,
			Conversation: title,
		}
	case "open":
		title := parts[1]
		conversation := c.data.Conversations[title]
		if conversation == nil {
			print("Conversation not found")
			return
		}
		c.conversation = title
		for _, message := range conversation.Messages {
			print(message.User + ": " + message.Body)
		}
	case "invite":
		userToSync <- &Tap{
			Type:         TYPE_INVITE,
			Conversation: c.conversation,
			Value:        parts[1],
		}
	case "send":
		userToSync <- &Tap{
			Type:         TYPE_MESSAGE,
			Conversation: c.conversation,
			Value:        strings.Join(parts[1:], " "),
		}
	case "close":
		c.conversation = ""
		print("")
	}

}

func printHelp() {
	print(`Available Commands:
  list: list conversations
  create <title> <participants>: create a conversation
  open <title>: open a conversation in the window
  invite <participant>: invite participants to a conversation
  say <message>: Say something in the current conversation
  close: close the current conversation
  leave <title>: leave a conversation
  exit: exit the program (and leave the current conversation)
  help: Show this help screen
`)
}

func print(message string) {
	// fmt.Print("\033[2J\033[1;1H\n\n\n\n\n\n\n\n\n\n\n\n\n\n$ ")
	// fmt.Print("\0337\033[1;1H", message, "\0338")
	fmt.Println(message)
}
