package main

// import (
// "fmt"
// "github.com/seanpont/gobro"
// _ "io/ioutil"
// "net"
// "sync"
// )
//
// type TcpTapServer struct {
// 	db *DB
// 	sync.Mutex
// }

// func NewTcpTapServer() *TcpTapServer {
// 	return &TcpTapServer{
// 		db: NewDB("server"),
// 	}
// }

// func (s *TcpTapServer) Serve(port string) {
// 	listener, err := net.Listen("tcp", ":"+port)
// 	gobro.CheckErr(err)
// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			continue
// 		}
// 		go s.handleConn(conn)
// 	}
// }

// func (s *TcpTapServer) handleConn(conn net.Conn) {
// 	s.handleTapChan(connToChan(conn))
// 	conn.Close()
// }

// func (s *TcpTapServer) handleTapChan(tapChan chan Tap) {
// 	tap := <-tapChan
// 	if tap.Type != "auth" || tap.User.Address == "" || tap.User.Name == "" {
// 		tapChan <- Tap{
// 			Type:    "error",
// 			Message: "First Tap must be of type 'auth' with address and name in user info",
// 		}
// 		return
// 	}
// 	fmt.Println("Recieved:", tap)
// 	for _, conversation := range s.db.GetConversations() {
// 		tapChan <- Tap{
// 			Type:         "sync",
// 			Conversation: conversation,
// 		}
// 	}

// }
