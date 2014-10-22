package main

import (
	"fmt"
	"github.com/seanpont/assert"
	"testing"
)

func TestServices(t *testing.T) {
	assert := assert.Assert(t)
	db := NewTempDB()
	defer db.Destroy()
	fmt.Println(db)
	edithId, err := db.CreateUser("Edith@circletech.com", "Edith Carmela")
	assert.Nil(err)
	assert.Equal(edithId, int64(1))
	edith := db.GetUserById(edithId)
	assert.NotNil(edith)
	assert.Equal(edith.Address, "edith@circletech.com")
	assert.Equal(edith.Name, "Edith Carmela")
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
