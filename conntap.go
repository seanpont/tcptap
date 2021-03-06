package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/seanpont/gobro"
	"github.com/seanpont/gobro/commander"
	"github.com/seanpont/gobro/strarr"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ===== MODEL ===============================================================

type Message struct {
	TapId int
	User  string
	Body  string
}

func (m *Message) String() string {
	return fmt.Sprintf("%s: %s", m.User, m.Body)
}

type Conversation struct {
	TapId    int
	Title    string
	Users    map[string]int
	Messages []*Message
}

func (c *Conversation) String() string {
	return c.Title
}

func (c *Conversation) LastMessage() *Message {
	last := len(c.Messages) - 1
	if last >= 0 {
		return c.Messages[last]
	} else {
		return &Message{}
	}
}

func (c *Conversation) NewMessage(tap *Tap) {
	c.Messages = append(c.Messages, &Message{
		TapId: tap.Id,
		User:  tap.User,
		Body:  tap.Value,
	})
}

type Data struct {
	Taps          []*Tap
	Users         map[string]int
	Conversations map[string]*Conversation
}

func NewData() *Data {
	return &Data{
		Taps:          make([]*Tap, 0),
		Users:         make(map[string]int),
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
	d.Users[tap.User] = tap.Id
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
		TapId:    tap.Id,
		Title:    title,
		Users:    make(map[string]int, 0),
		Messages: make([]*Message, 0),
	}
	c.Users[tap.User] = tap.Id
	for _, user := range tap.Args {
		d.Users[user] = tap.Id
		c.Users[user] = tap.Id
	}
	if tap.Value == "" {
		firstMessage := "Created conversation"
		if len(tap.Args) > 0 {
			firstMessage += " with " + strings.Join(tap.Args, ", ")
		}
		tap.Value = firstMessage
	}
	c.NewMessage(tap)
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
	c.NewMessage(tap)
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
		d.Users[user] = tap.Id
		c.Users[user] = tap.Id
	}
	tap.Value = fmt.Sprintf("%s invited %s", tap.User, strings.Join(tap.Args, ", "))
	c.NewMessage(tap)
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
	commander.CheckArgs(args, 1, "Usage: tcptap streamTapServer <port>")
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
		tap.Id = len(s.data.Taps)
		fmt.Println("Processing: ", tap)
		err := s.data.Update(tap)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		s.data.Taps = append(s.data.Taps, tap)

		for user, tapChan := range s.tapChans {
			if s.isRelevant(user, tap) {
				fmt.Printf("Sending %s to %s\n", tap.Type, user)
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
					if s.isInvitingUser(user, tap) {
						s.replayConversation(tap, outbox)
					}
					outbox <- tap
				}
			}
		}
	}
}

func (s *ConnTapServer) isInvitingUser(user string, tap *Tap) bool {
	return tap.Type == TYPE_INVITE && strarr.Contains(tap.Args, user)
}

func (s *ConnTapServer) replayConversation(inviteTap *Tap, outbox chan<- *Tap) {
	fmt.Printf("Replaying conversation: %s\n", inviteTap.Conversation)
	for tapCursor := 0; tapCursor < inviteTap.Id; tapCursor++ {
		tap := s.data.Taps[tapCursor]
		if tap.Conversation == inviteTap.Conversation {
			fmt.Printf("Replay: %s\n", tap.Type)
			outbox <- tap
		}
	}
}

func (s *ConnTapServer) isRelevant(user string, tap *Tap) bool {
	switch tap.Type {
	case TYPE_AUTH:
		return true
	case TYPE_CONVERSATION, TYPE_MESSAGE, TYPE_INVITE:
		// User must be in conversation AND must have been joined prior to this tap
		membershipId := s.data.Conversations[tap.Conversation].Users[user]
		return membershipId > 0 && membershipId <= tap.Id
	default:
		return false
	}
}

func notify(tapChan chan<- bool) {
	select {
	case tapChan <- true:
	default:
	}
}

// ===== Client ==============================================================

func connTapClient(args []string) {
	commander.CheckArgs(args, 1, "Usage: tcptap connTapClient <host:port>")
	name, _ := commander.Prompt("Please enter your name: ")
	NewConnTapClient(name).connect(args[0])
}

type ConnTapClient struct {
	user           string
	data           *Data
	conversation   *Conversation
	userToSync     chan *Tap
	syncToUser     chan *Tap
	isViewingUsers bool
	isViewingHelp  bool
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
	defer close(c.syncToUser)

	// Authentication
	outbox <- NewTap(TYPE_AUTH, c.user, "", "0")

	// Listen loop
	for {
		select {
		case tap, ok := <-inbox:
			if !ok {
				return
			}
			// fmt.Printf("%s received: %s\n", c.user, tap.Type)
			err := c.data.Update(tap)
			if err != nil {
				// fmt.Printf("%s encountered error processing %s: %s\n",
				// c.user, tap.Type, err.Error())
				continue
			}
			c.syncToUser <- tap
		case tap, ok := <-c.userToSync:
			if !ok {
				return
			}
			outbox <- tap
		}
	}
}

func (c *ConnTapClient) handle() {
	defer close(c.userToSync)

	prompt := make(chan string)
	go func() {
		defer close(prompt)
		reader := bufio.NewReader(os.Stdin)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				return
			}
			prompt <- string(line)
		}
	}()

	c.printHelp(true)

	for {
		select {
		case _, ok := <-c.syncToUser:
			if !ok {
				c.print("Server has closed connection")
				return
			}
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
	c.isViewingUsers = false
	c.isViewingHelp = false

	parts := strings.SplitN(message, " ", 2)
	cmd := parts[0]
	val := ""
	if len(parts) == 2 {
		val = parts[1]
	}

	if c.conversation == nil {
		switch cmd {
		case "help":
			c.printHelp(true)
		case "exit":
			os.Exit(0)
		case "users":
			c.printUsers(true)
		case "inbox":
			c.printInbox(true)
		case "create":
			c.createConversation(val)
			c.printInbox(true)
		case "open":
			c.openConversation(val)
		default:
			c.printHelp(true)
		}
	} else {
		switch cmd {
		case "help":
			c.printHelp(true)
		case "exit":
			os.Exit(0)
		case "users":
			c.printUsers(true)
		case "invite":
			c.inviteUsers(val)
			c.printMessages(true)
		case "close":
			c.conversation = nil
			c.printInbox(true)
		default:
			c.userToSync <- &Tap{
				Type:         TYPE_MESSAGE,
				Conversation: c.conversation.Title,
				Value:        message,
			}
			c.printMessages(true)
		}
	}
}

func (c *ConnTapClient) inviteUsers(args string) {
	users := strings.Split(args, ",")
	strarr.TrimAll(users)
	c.userToSync <- &Tap{
		Type:         TYPE_INVITE,
		Conversation: c.conversation.Title,
		Args:         users,
	}
}

func (c *ConnTapClient) createConversation(args string) {
	titleAndUsers := strings.SplitN(args, ":", 2)
	title := strings.Trim(titleAndUsers[0], " ")
	var users []string
	if len(titleAndUsers) == 2 {
		users = strings.Split(titleAndUsers[1], ",")
		strarr.TrimAll(users)
	}

	c.userToSync <- &Tap{
		Type:         TYPE_CONVERSATION,
		Conversation: title,
		Args:         users,
	}
}

func (c *ConnTapClient) openConversation(title string) {
	c.conversation = c.data.Conversations[title]
	if c.conversation == nil {
		c.print("Conversation %s not found", title)
	} else {
		c.printMessages(true)
	}
}

func (c *ConnTapClient) updateView() {
	if c.isViewingUsers {
		c.printUsers(false)
	} else if c.isViewingHelp {
		// do nothing
	} else if c.conversation == nil {
		c.printInbox(false)
	} else {
		c.printMessages(false)
	}
}

func (c *ConnTapClient) printInbox(clearView bool) {
	inbox := make([]string, 0, 20)
	for title, conversation := range c.data.Conversations {
		inbox = append(inbox, title+"\n  "+conversation.LastMessage().String())
		if len(inbox) == 18 {
			break
		}
	}
	content := strings.Join(inbox, "\n")
	if clearView {
		c.print(content)
	} else {
		c.updateContent(content)
	}
}

func (c *ConnTapClient) printMessages(clearView bool) {
	messages := make([]string, 0, 20)
	start := len(c.conversation.Messages) - 20
	start = gobro.Max(start, 0)
	for _, message := range c.conversation.Messages[start:] {
		messages = append(messages, message.String())
	}
	content := strings.Join(messages, "\n")
	if clearView {
		c.print(content)
	} else {
		c.updateContent(content)
	}
}

func (c *ConnTapClient) printUsers(clearView bool) {
	c.isViewingUsers = true
	header := "All users:"
	userSet := c.data.Users
	users := make([]string, 0, len(c.data.Users))
	if c.conversation != nil {
		userSet = c.conversation.Users
		header = c.conversation.Title + " users:"
	}
	for user, _ := range userSet {
		users = append(users, user)
	}
	content := fmt.Sprintf("%s\n  %s", header, strings.Join(users, "\n  "))
	if clearView {
		c.print(content)
	} else {
		c.updateContent(content)
	}

}

func (c *ConnTapClient) printHelp(clearView bool) {
	c.isViewingHelp = true
	content := `Available Commands:

  From the inbox:
    inbox: show the inbox
    create <title> [:<participants>, ...]: create a conversation
    	To include participants, put ':' followed by comma-separated list of users.
    	For example:
    	$ create Good Apples: John Apple, Fred Pear, Bob Watermelon
    open <title>: open a conversation in the window
    users: show all users
    leave <title>: leave a conversation
  From a conversation:
    users: show users in conversation
    invite <participants>: invite list of comma-separated participants to conversation
    close: close the current conversation (go back to the inbox)
    <message>: Say something in the current conversation
  From anywhere:
    exit: exit the program (and leave the current conversation)
    help: Show this help screen
`
	if clearView {
		c.print(content)
	} else {
		c.updateContent(content)
	}
}

func (c *ConnTapClient) print(format string, a ...interface{}) {
	content := fmt.Sprintf(format, a...)
	// ensure that it is 20 lines long
	numLines := strings.Count(content, "\n")
	for i := numLines; i < 20; i++ {
		content += "\n"
	}

	header := "Inbox"
	if c.conversation != nil {
		header = c.conversation.Title
	}
	divider := "\n================================\n"
	prompt := c.user + "$ "

	fmt.Print("\033[2J\033[1;1H" + header + divider + content + "\n" + prompt)
}

func (c *ConnTapClient) updateContent(format string, a ...interface{}) {
	content := fmt.Sprintf(format, a...)

	header := "Inbox"
	if c.conversation != nil {
		header = c.conversation.Title
	}
	divider := "\n================================\n"

	fmt.Print("\033[s\033[1;1H" +
		strings.Repeat("\033[K\033[1B", 22) +
		"\033[1;1H" + header + divider + content + "\033[u")
}
