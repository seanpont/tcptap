package main

import (
	"errors"
	"fmt"
	"github.com/seanpont/gobro"
	"os"
	"sync"
)

// ===== MODEL ===============================================================

type User struct {
	Name    string
	Address string
}

type Message struct {
	User *User
	Body string
}

type Conversation struct {
	Users    []*User
	Messages []*Message
	Title    string
}

// ===== IN-MEMORY DB ========================================================

type DB struct {
	Users         []*User
	Conversations []*Conversation
	sync.Mutex
}

var db DB

func init() {
	db = DB{
		Users:         make([]*User, 0, 100),
		Conversations: make([]*Conversation, 0, 100),
	}
}

// ===== CONTROLLERS =========================================================

func CreateUser(name, address string) (*User, error) {
	if address == "" {
		return nil, errors.New("Address Required")
	}
	db.Lock()
	defer db.Unlock()
	for _, u := range db.Users {
		if u.Address == address {
			u.Name = name
			return u, nil
		}
	}
	u := &User{Name: name, Address: address}
	db.Users = append(db.Users, u)
	return u, nil
}

func CreateConversation(title string, creator *User, participants ...*User) *Conversation {
	c := &Conversation{
		Users:    make([]*User, 0, len(participants)+1),
		Messages: make([]*Message, 0),
	}
	c.Users = append(c.Users, creator)
	c.Users = append(c.Users, participants...)

	message := &Message{
		User: creator,
		Body: creator.Name + " Created the conversation " + title,
	}
	c.Messages = append(c.Messages, message)

	c.Title = title

	db.Lock()
	defer db.Unlock()
	db.Conversations = append(db.Conversations, c)
	return c
}

// ===== NETWORKING LAYER ====================================================

func Server(args []string) {
	fmt.Fprintln(os.Stderr, "UNIMPLEMENTED")
}

func Client(args []string) {
	fmt.Fprintln(os.Stderr, "UNIMPLEMENTED")
}

func main() {
	gobro.NewCommandMap(Server, Client).Run(os.Args)
}
