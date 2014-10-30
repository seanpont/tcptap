package main

import (
	_ "fmt"
	"github.com/seanpont/assert"
	"testing"
	"time"
)

func connect(server *ConnTapServer, user string) (client *ConnTapClient) {
	client = NewConnTapClient(user)
	clientToServer := make(chan *Tap, 3)
	serverToClient := make(chan *Tap, 3)
	go client.sync(serverToClient, clientToServer)
	go server.handle(clientToServer, serverToClient)
	return client
}

func drain(count int, client *ConnTapClient) bool {
	for count > 0 {
		select {
		case <-client.syncToUser:
			count--
		case <-time.After(time.Millisecond * 10):
			return false
		}
	}
	return true
}

func TestReplayConversation(t *testing.T) {
	assert := assert.Assert(t)
	server := NewConnTapServer()

	sean := connect(server, "sean")
	alex := connect(server, "alex")
	sean.userToSync <- NewTap(TYPE_CONVERSATION, "sean", "title", "")
	sean.userToSync <- NewTap(TYPE_MESSAGE, "sean", "title", "message1")
	assert.True(drain(4, sean), "")
	assert.True(drain(2, alex), "")
	sean.userToSync <- NewTap(TYPE_INVITE, "sean", "title", "", "alex")
	assert.True(drain(1, sean), "")
	assert.True(drain(3, alex), "")

	assert.NotNil(sean.data.Conversations["title"])
	assert.NotNil(alex.data.Conversations["title"])
}

func TestIsRelevant(t *testing.T) {
	assert := assert.Assert(t)
	server := NewConnTapServer()

	sean := connect(server, "sean")
	alex := connect(server, "alex")

	// both clients receive each other's auths
	assert.True(drain(2, sean), "")
	assert.True(drain(2, alex), "")

	// sean creates a conversation that includes John but not alex
	sean.userToSync <- NewTap(TYPE_CONVERSATION, "sean", "apples", "tasty", "john")
	assert.True(drain(1, sean), "")
	assert.Equal(len(sean.data.Conversations["apples"].Users), 2)
	assert.False(server.data.Conversations["apples"].Users["alex"] > 0, "")

	// alex does not get the conversation
	assert.False(drain(1, alex), "")
	assert.Equal(len(alex.data.Conversations), 0)

	// but john will
	john := connect(server, "john")
	assert.True(drain(4, john), "") // two auths, conversation, auth
	assert.True(drain(1, sean), "") // john's auth
	assert.True(drain(1, alex), "") // john's auth

	// John is now all caught up
	assert.NotNil(john.data.Conversations["apples"])

	// john and sean chat about apples
	john.userToSync <- NewTap(TYPE_MESSAGE, "john", "apples", "hi")
	sean.userToSync <- NewTap(TYPE_MESSAGE, "sean", "apples", "hello")
	assert.True(drain(2, sean), "")
	assert.True(drain(2, john), "")
	assert.False(drain(2, alex), "") // alex doesn't get anything

	// Both users should have 3 messages (initial, john's, and sean's)
	assert.Equal(len(john.data.Conversations["apples"].Messages), 3)
	assert.Equal(len(sean.data.Conversations["apples"].Messages), 3)

	// Now john invites alex
	john.userToSync <- NewTap(TYPE_INVITE, "john", "apples", "", "alex")
	// alex should now receive all taps about conversation, in order, including his own invite
	assert.True(drain(4, alex), "") // conversation, hi, hello, invite
	assert.True(drain(1, sean), "") // invite
	assert.True(drain(1, john), "") // invite

	// and now alex is all caught up
	assert.NotNil(alex.data.Conversations["apples"])
}

func TestConnTap(t *testing.T) {
	assert := assert.Assert(t)
	server := NewConnTapServer()

	sean := connect(server, "sean")

	// Sean should get his auth back
	authTap := <-sean.syncToUser
	assert.NotNil(authTap)
	// And users should have been created on client and server
	assert.Equal(len(sean.data.Users), 1)
	assert.Equal(len(server.data.Users), 1)

	//Create a conversation
	sean.userToSync <- NewTap(TYPE_CONVERSATION, "sean", "bananas", "Hey guys", "alex", "will")
	conversationTap := <-sean.syncToUser
	assert.NotNil(conversationTap)

	// Client and server should both have conversation with 3 participants and 1 message
	assert.Equal(len(sean.data.Conversations), 1)
	conversation := sean.data.Conversations["bananas"]
	assert.Equal(len(conversation.Users), 3) // sean, alex, will
	assert.Equal(len(sean.data.Users), 3)
	assert.Equal(len(conversation.Messages), 1)
	assert.Equal(conversation.Messages[0].Body, "Hey guys")

	assert.Equal(len(server.data.Conversations), 1)
	conversation = server.data.Conversations["bananas"]
	assert.Equal(len(conversation.Users), 3) // sean, alex, will
	assert.Equal(len(conversation.Messages), 1)

	// Alex joins the party
	alex := connect(server, "alex")
	drain(3, alex) // sean auth, cconversation, alex auth
	assert.Equal(len(alex.data.Conversations), 1)
	alex.userToSync <- NewTap(TYPE_MESSAGE, "alex", "bananas", "Hey Sean")
	drain(1, alex) // message
	drain(2, sean) // alex auth, message
	assert.Equal(sean.data.Conversations["bananas"].Messages[1].Body, "Hey Sean")
}
