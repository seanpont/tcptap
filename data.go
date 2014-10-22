package main

import (
	"errors"
	_ "fmt"
	"github.com/mxk/go-sqlite/sqlite3"
	"github.com/seanpont/gobro"
	"os"
	"strings"
)

// ===== MODEL ===============================================================

type User struct {
	Id      int64
	Address string
	Name    string
	Avatar  string
	Version int
}

type Conversation struct {
	Id      int64
	Title   string
	Version int
}

type Participant struct {
	ConversationId int64
	UserId         int64
	Version        int
}

type Message struct {
	Id             int64
	ConversationId int64
	UserId         int64
	Type           string
	Body           string
}

// ===== INITIALIZATION ======================================================

type DB struct {
	fname string
}

func (db *DB) Conn() *sqlite3.Conn {
	conn, err := sqlite3.Open(db.fname)
	gobro.CheckErr(err)
	return conn
}

func (db *DB) init() {
	conn := db.Conn()
	defer conn.Close()
	err := conn.Exec(`CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    address TEXT UNIQUE, 
    name TEXT, 
    avatar TEXT,
    version INTEGER DEFAULT 1
  )`)
	gobro.CheckErr(err)
}

func (db *DB) Destroy() {
	os.Remove(db.fname)
}

func NewDB(fname string) *DB {
	db := &DB{
		fname: fname,
	}
	db.init()
	return db
}

func NewTempDB() *DB {
	fname := os.TempDir() + "tcptap.sqlite"
	os.Remove(fname)
	return NewDB(fname)
}

// ===== User ================================================================

func (db *DB) CreateUser(address, name string) (userId int64, err error) {
	if address == "" {
		return userId, errors.New("Address required")
	}
	address = strings.ToLower(address)
	user := User{
		Address: strings.ToLower(address),
		Name:    name,
	}
	conn := db.Conn()
	defer conn.Close()
	err = conn.Exec(
		"INSERT INTO users(address, name, avatar) VALUES (?, ?, ?)",
		user.Address, user.Name, user.Avatar)
	userId = conn.LastInsertId()
	return
}

func (db *DB) GetUserById(userId int64) *User {
	conn := db.Conn()
	defer conn.Close()
	stmt, err := conn.Query(
		"SELECT id, address, name, avatar, version FROM users WHERE id = ?", userId)
	gobro.CheckErr(err)
	var user User
	stmt.Scan(&user.Id, &user.Address, &user.Name, &user.Avatar, &user.Version)
	return &user
}
