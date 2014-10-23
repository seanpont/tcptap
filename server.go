package main

import (
	"encoding/json"
	"fmt"
	"github.com/seanpont/gobro"
	_ "io/ioutil"
	"net"
	"sync"
)

type TcpTapServer struct {
	tapChan chan Tap
	conns   []net.Conn
	sync.Mutex
}

func NewTcpTapServer() *TcpTapServer {
	server := new(TcpTapServer)
	server.tapChan = make(chan Tap)
	server.conns = make([]net.Conn, 0)
	return server
}

func (s *TcpTapServer) Serve(port string) {
	go s.deliverTaps()
	listener, err := net.Listen("tcp", ":"+port)
	gobro.CheckErr(err)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *TcpTapServer) deliverTaps() {
	for {
		tap := <-s.tapChan
		fmt.Println("Deliver:", tap)
	}
}

func (s *TcpTapServer) handleConn(conn net.Conn) {
	defer s.removeConn(conn)
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	var tap Tap
	decoder.Decode(&tap)
	if tap.Type != "auth" || tap.Message == "" || tap.Message != "5525" {
		encoder.Encode(Tap{
			Type:    "error",
			Sender:  "system",
			Message: "First message type must be 'auth' with address in body",
		})
		return
	}
	address := tap.Message
	s.addConn(conn)
	for {
		err := decoder.Decode(&tap)
		if err != nil {
			s.tapChan <- Tap{
				Type:    "removed",
				Sender:  address,
				Message: address + " has left the conversation",
			}
			return
		}
		tap.Sender = address
		tap.Type = "message"
		s.tapChan <- tap
	}
}

func (s *TcpTapServer) addConn(conn net.Conn) {
	s.Lock()
	defer s.Unlock()
	s.conns = append(s.conns, conn)
}

func (s *TcpTapServer) removeConn(conn net.Conn) {
	conn.Close()
	s.Lock()
	defer s.Unlock()
	for i, c := range s.conns {
		if c == conn {
			end := len(s.conns) - 1
			s.conns[i], s.conns[end], s.conns = s.conns[end], nil, s.conns[:end]
			return
		}
	}

}
