package main

import (
	_ "fmt"
	"github.com/seanpont/assert"
	"testing"
)

func TestDB(t *testing.T) {
	assert := assert.Assert(t)
	db := NewDB("test")
	defer db.Destroy()
	edithId, err := db.InsertUser(User{
		Address: "Edith@circletech.com",
		Name:    "Edith Carmela",
	})
	assert.Nil(err)
	assert.Equal(edithId, int64(1))
	edith := db.GetUserById(edithId)
	assert.NotNil(edith)
	assert.True(edith.Id > 0, "Id > 0")
	assert.Equal(edith.Address, "edith@circletech.com")
	assert.Equal(edith.Name, "Edith Carmela")

	conversations := db.GetConversations()
	assert.Equal(len(conversations), 0)

	conversationId, err := db.InsertConversation(Conversation{
		Title: "foobar",
	})
	assert.Nil(err)
	assert.True(conversationId > 0, "conversationId > 0")
	conversations = db.GetConversations()
	assert.Equal(len(conversations), 1)

	// roland, _ := db.CreateUser("Roland Sawyer", "roland@circletech.com")
	// jackie, _ := db.CreateUser("Jackie Richards", "jackie@circletech.com")

	// conversation := CreateConversation("Friends", edith, roland, jackie)
	// assert.NotNil(conversation)
	// assert.Equal(len(conversation.Users), 3)
	// assert.Equal(conversation.Users[0], edith)
	// assert.Equal(conversation.Title, "Friends")
	// assert.Equal(len(conversation.Messages), 1)
	// assert.Equal(conversation.Messages[0].User, edith)
	// assert.Equal(conversation.ID, 1)

	// SendMessage(conversation, roland, "Hi guys")
	// SendMessage(conversation, jackie, "Hey y'all")
	// SendMessage(conversation, edith, "👍")
	// assert.Equal(len(conversation.Messages), 4)
	// assert.Equal(conversation.Messages[3].Body, "👍")

}
