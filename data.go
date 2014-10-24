package main

// import (
// 	"errors"
// 	_ "fmt"
// 	"github.com/mxk/go-sqlite/sqlite3"
// 	"github.com/seanpont/gobro"
// 	"os"
// 	"strings"
// )

// // ===== MODEL ===============================================================

// type User struct {
// 	Id      int64
// 	Address string
// 	Name    string
// }

// type Conversation struct {
// 	Id    int64
// 	Title string
// }

// type Participant struct {
// 	ConversationId int64
// 	UserId         int64
// }

// type Message struct {
// 	Id             int64
// 	ConversationId int64
// 	UserId         int64
// 	Type           string
// 	Body           string
// }

// // ===== INITIALIZATION ======================================================

// type DB struct {
// 	fname string
// }

// func (db *DB) Conn() *sqlite3.Conn {
// 	conn, err := sqlite3.Open(db.fname)
// 	gobro.CheckErr(err)
// 	return conn
// }

// func (db *DB) init() {
// 	conn := db.Conn()
// 	defer conn.Close()
// 	err := conn.Exec(`CREATE TABLE IF NOT EXISTS users (
//     id INTEGER PRIMARY KEY AUTOINCREMENT,
//     address TEXT UNIQUE,
//     name TEXT
//   )`)
// 	gobro.CheckErr(err)
// 	err = conn.Exec(`CREATE TABLE IF NOT EXISTS conversations (
//     id INTEGER PRIMARY KEY AUTOINCREMENT,
//     title TEXT UNIQUE
//   )`)
// 	gobro.CheckErr(err)
// }

// func (db *DB) Destroy() {
// 	os.Remove(db.fname)
// }

// func NewDB(fname string) *DB {
// 	fname = os.TempDir() + fname + ".sqlite"
// 	os.Remove(fname)
// 	db := &DB{
// 		fname: fname,
// 	}
// 	db.init()
// 	return db
// }

// // ===== User ================================================================

// func (db *DB) InsertUser(user User) (userId int64, err error) {
// 	if user.Address == "" {
// 		return userId, errors.New("Address required")
// 	}
// 	user.Address = strings.ToLower(user.Address)
// 	conn := db.Conn()
// 	defer conn.Close()
// 	err = conn.Exec(
// 		"INSERT OR REPLACE INTO users(address, name) VALUES (?, ?)",
// 		user.Address, user.Name)
// 	userId = conn.LastInsertId()
// 	return
// }

// func (db *DB) GetUserById(userId int64) *User {
// 	conn := db.Conn()
// 	defer conn.Close()
// 	stmt, err := conn.Query(
// 		"SELECT id, address, name FROM users WHERE id = ?", userId)
// 	gobro.CheckErr(err)
// 	var user User
// 	stmt.Scan(&user.Id, &user.Address, &user.Name)
// 	return &user
// }

// // ===== CONVERSATIONS =======================================================

// func (db *DB) GetConversations() []Conversation {
// 	conn := db.Conn()
// 	defer conn.Close()
// 	conversations := make([]Conversation, 0)
// 	stmt, err := conn.Query("SELECT id, title FROM conversations")
// 	for ; err == nil; stmt.Next() {
// 		var conversation Conversation
// 		stmt.Scan(&conversation.Id, &conversation.Title)
// 		conversations = append(conversations, conversation)
// 	}
// 	return conversations
// }

// func (db *DB) InsertConversation(conversation Conversation) (conversationId int64, err error) {
// 	conn := db.Conn()
// 	defer conn.Close()
// 	err = conn.Exec(
// 		"INSERT OR IGNORE INTO conversations(title) VALUES (?)", conversation.Title)
// 	conversationId = conn.LastInsertId()
// 	return
// }
