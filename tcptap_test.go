package main

import (
	"fmt"
	"github.com/seanpont/assert"
	"testing"
)

func TestStreamTap(t *testing.T) {
	assert := assert.Assert(t)

	server := NewConnTapServer()
	client := NewConnTapClient("sean")

	// wire them up!
	clientToServer := make(chan *Tap)
	serverToClient := make(chan *Tap)
	syncToUser := make(chan *Tap)
	userToSync := make(chan *Tap)

	go server.handle(clientToServer, serverToClient)
	go client.sync(serverToClient, clientToServer, userToSync, syncToUser)

	// We should get our auth back
	authTap := <-syncToUser
	fmt.Println(authTap)
	assert.NotNil(authTap)

	//Create a conversation
	userToSync <- NewTap(TYPE_CONVERSATION, "", "bananas", "", "alex", "", "will", "")
	conversationTap := <-syncToUser
	fmt.Println(conversationTap)
	assert.NotNil(conversationTap)

	assert.Equal(len(server.data.Conversations), 1)
	assert.Equal(len(client.data.Conversations["bananas"].Users), 3) // sean, alex, will

	assert.Equal(len(client.data.Conversations), 1)
	assert.Equal(len(client.data.Conversations["bananas"].Users), 3) // sean, alex, will

}
