package main

import (
	"encoding/json"
	"errors"
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

func (m *Message) String() string {
	return fmt.Sprintf("%s: %s", m.User, m.Body)
}

type Conversation struct {
	Title    string
	Users    map[string]bool
	Messages []*Message
}

func (c *Conversation) String() string {
	return c.Title
}

func (c *Conversation) LastMessage() string {
	last := len(c.Messages) - 1
	if last >= 0 {
		return c.Messages[last].String()
	} else {
		return ""
	}
}

func (c *Conversation) NewMessage(user, message string) {
	c.Messages = append(c.Messages, &Message{User: user, Body: message})
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

func (d *Data) Update(tap *Tap) (err error) {
	switch tap.Type {
	case TYPE_AUTH:
		err = d.CreateUser(tap)
	case TYPE_CONVERSATION:
		err = d.CreateConversation(tap)
	case TYPE_MESSAGE:
		err = d.SendMessage(tap)
	case TYPE_INVITE:
		err = d.Invite(tap)
	}
	return
}

func (d *Data) CreateUser(tap *Tap) error {
	if tap.User == "" {
		return errors.New("User name required")
	}
	d.Users[tap.User] = true
	return nil
}

func (d *Data) CreateConversation(tap *Tap) error {
	title := tap.Conversation
	if title == "" {
		return errors.New("Conversation title required")
	}
	if d.Conversations[title] != nil {
		return errors.New("Conversation '" + title + "' already exists")
	}
	c := &Conversation{
		Title:    title,
		Users:    make(map[string]bool, 0),
		Messages: make([]*Message, 0),
	}
	c.Users[tap.User] = true
	for _, user := range tap.Args {
		d.Users[user] = true
		c.Users[user] = true
	}
	if tap.Value != "" {
		c.NewMessage(tap.User, tap.Value)
	}
	d.Conversations[c.Title] = c
	return nil
}

func (d *Data) SendMessage(tap *Tap) error {
	if tap.Conversation == "" || tap.Value == "" {
		return errors.New("Conversation and Value (message body) required")
	}
	c := d.Conversations[tap.Conversation]
	if c == nil {
		return errors.New("Conversation '" + tap.Conversation + "' not found")
	}
	c.NewMessage(tap.User, tap.Value)
	return nil
}

func (d *Data) Invite(tap *Tap) error {
	if tap.Conversation == "" || len(tap.Args) == 0 {
		return errors.New("Conversation and args (new participants) required")
	}
	c := d.Conversations[tap.Conversation]
	if c == nil {
		return errors.New("Conversation '" + tap.Conversation + "' not found")
	}
	for _, user := range tap.Args {
		d.Users[user] = true
		c.Users[user] = true
	}
	c.NewMessage(tap.User,
		fmt.Sprintf("%s invited %s", tap.User, strings.Join(tap.Args, ", ")))
	return nil
}

// ===== TAP PROTOCOL ========================================================

type Tap struct {
	Id           int      `json:"id"`
	Type         string   `json:"type"`
	User         string   `json:"user"`
	Conversation string   `json:"conversation"`
	Value        string   `json:"value"`
	Args         []string `json:"args"`
}

func NewTap(_type, user, conversation, value string, args ...string) *Tap {
	tap := Tap{
		Type:         _type,
		User:         user,
		Conversation: conversation,
		Value:        value,
		Args:         args,
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
		err := s.data.Update(tap)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

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
	tapIdStr := authTap.Value
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

// ===== Client ==============================================================

func connTapClient(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap connTapClient <host:port>")
	name, _ := gobro.Prompt("Please enter your name: ")
	NewConnTapClient(name).connect(args[0])
}

type ConnTapClient struct {
	user         string
	data         *Data
	conversation *Conversation
	userToSync   chan *Tap
	syncToUser   chan *Tap
}

func NewConnTapClient(user string) *ConnTapClient {
	return &ConnTapClient{
		user:       user,
		data:       NewData(),
		userToSync: make(chan *Tap),
		syncToUser: make(chan *Tap),
	}
}

func (c *ConnTapClient) connect(service string) {
	c.print("Connecting...")
	conn, err := net.Dial("tcp", service)
	gobro.CheckErr(err)
	defer conn.Close()
	c.print("Connected!")

	inbox, outbox := connToChan(conn)
	go c.sync(inbox, outbox)
	c.handle()
}

func (c *ConnTapClient) sync(inbox <-chan *Tap, outbox chan<- *Tap) {
	defer close(outbox)

	// Authentication
	outbox <- NewTap(TYPE_AUTH, c.user, "", "0")

	// Listen loop
	for {
		select {
		case tap, ok := <-inbox:
			if !ok {
				return
			}
			// fmt.Printf("%s received: %v\n", c.user, tap)
			err := c.data.Update(tap)
			if err != nil {
				c.print("%s encountered error while updating data: %d", err)
				return
			}
			c.syncToUser <- tap
		case tap, ok := <-c.userToSync:
			if !ok {
				c.print("userToSync closed")
				return
			}
			outbox <- tap
		}
	}
}

func (c *ConnTapClient) handle() {
	defer func() {
		c.print("handle: close userToSync")
		close(c.userToSync)
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

	c.printHelp()

	for {
		select {
		case _ = <-c.syncToUser:
			c.updateView()
		case cmd, ok := <-prompt:
			if !ok {
				c.print("Goodbye")
				return
			}
			c.handleCmd(cmd)
		}
	}
}

func (c *ConnTapClient) handleCmd(message string) {
	parts := strings.SplitN(message, " ", 2)
	cmd := parts[0]
	val := ""
	if len(parts) == 2 {
		val = parts[1]
	}
	args := strings.Split(message, " ")[1:]
	switch cmd {
	case "help":
		c.printHelp()
	case "inbox":
		c.conversation = nil
		c.printInbox()
	case "users":
		c.printUsers()
	case "create":
		c.userToSync <- &Tap{
			Type:         TYPE_CONVERSATION,
			Conversation: val,
		}
		c.updateView()
	case "open":
		c.openConversation(val)
	case "invite":
		if c.conversation == nil {
			c.print("You must open a conversation before you can invite users")
			return
		}
		c.userToSync <- &Tap{
			Type:         TYPE_INVITE,
			Conversation: c.conversation.Title,
			Args:         args,
		}
	case "send", "say":
		if c.conversation == nil {
			c.print("You must open a conversation to send a message")
			return
		}
		c.userToSync <- &Tap{
			Type:         TYPE_MESSAGE,
			Conversation: c.conversation.Title,
			Value:        val,
		}
		c.updateView()
	case "close":
		c.conversation = nil
		c.updateView()
	case "exit":
		os.Exit(0)
	default:
		if c.conversation != nil {
			c.userToSync <- &Tap{
				Type:         TYPE_MESSAGE,
				Conversation: c.conversation.Title,
				Value:        message,
			}
			c.printMessages()
		} else {
			c.printHelp()
		}
	}
}

func (c *ConnTapClient) openConversation(title string) {
	c.conversation = c.data.Conversations[title]
	if c.conversation == nil {
		c.print("Conversation %s not found", title)
	} else {
		c.printMessages()
	}
}

func (c *ConnTapClient) updateView() {
	if c.conversation == nil {
		c.printInbox()
	} else {
		c.printMessages()
	}
}

func (c *ConnTapClient) printInbox() {
	inbox := make([]string, 0, len(c.data.Conversations))
	for title, conversation := range c.data.Conversations {
		inbox = append(inbox, title+"\n  "+conversation.LastMessage())
	}
	c.print(strings.Join(inbox, "\n"))
}

func (c *ConnTapClient) printMessages() {
	messages := make([]string, 0, len(c.conversation.Messages))
	for _, message := range c.conversation.Messages {
		messages = append(messages, message.String())
	}
	c.print(strings.Join(messages, "\n"))
}

func (c *ConnTapClient) printUsers() {
	header := "All Users:"
	userSet := c.data.Users
	users := make([]string, 0, len(c.data.Users))
	if c.conversation != nil {
		userSet = c.conversation.Users
		header = c.conversation.Title + " Users:"
	}
	for user, _ := range userSet {
		users = append(users, user)
	}
	c.print("%s\n\n%s", header, strings.Join(users, "\n"))
}

func (c *ConnTapClient) printHelp() {
	c.print(`Available Commands:
  inbox: go to inbox
  create <title>: create a conversation
  open <title>: open a conversation in the window
  users: print users. If a conversation is open, print users in conversation
  invite <participants>: invite participants to a conversation
  say <message>: Say something in the current conversation
  close: close the current conversation
  leave <title>: leave a conversation
  exit: exit the program (and leave the current conversation)
  help: Show this help screen
`)
}

func (c *ConnTapClient) print(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	curtitle := "inbox"
	if c.conversation != nil {
		curtitle = c.conversation.Title
	}

	fmt.Print("\033[2J\033[1;1H\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n" +
		curtitle +
		"$ ")
	fmt.Print("\0337\033[1;1H", message, "\0338")
}
