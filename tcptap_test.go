package main

import (
	"fmt"
	"github.com/seanpont/assert"
	"testing"
)

func TestServices(t *testing.T) {
	assert := assert.Assert(t)
	fmt.Println("TestServices")
	edith, _ := CreateUser("Edith Carmela", "edith@circletech.com")
	roland, _ := CreateUser("Roland Sawyer", "roland@circletech.com")
	jackie, _ := CreateUser("Jackie Richards", "jackie@circletech.com")

	conversation := CreateConversation("Friends", edith, roland, jackie)
	assert.NotNil(conversation)
	assert.Equal(len(conversation.Users), 3)
	assert.Equal(conversation.Users[0], edith)
	assert.Equal(conversation.Title, "Friends")
	assert.Equal(len(conversation.Messages), 1)
	assert.Equal(conversation.Messages[0].User, edith)
}
