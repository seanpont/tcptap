package main

import (
	_ "fmt"
	"github.com/seanpont/assert"
	"testing"
)

func connect(server *ConnTapServer, user string) (client *ConnTapClient) {
	client = NewConnTapClient(user)
	clientToServer := make(chan *Tap)
	serverToClient := make(chan *Tap)
	go client.sync(serverToClient, clientToServer)
	go server.handle(clientToServer, serverToClient)
	return client
}

func drain(tapChan chan *Tap, count int) {
	for i := 0; i < count; i++ {
		_ = <-tapChan
	}
}

func TestStreamTap(t *testing.T) {
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
	drain(alex.syncToUser, 3) // sean auth, cconversation, alex auth
	assert.Equal(len(alex.data.Conversations), 1)
	alex.userToSync <- NewTap(TYPE_MESSAGE, "alex", "bananas", "Hey Sean")
	drain(alex.syncToUser, 1) // message
	drain(sean.syncToUser, 2) // alex auth, message
	assert.Equal(sean.data.Conversations["bananas"].Messages[1].Body, "Hey Sean")
}
