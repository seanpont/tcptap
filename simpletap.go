package main

import (
	"fmt"
	"github.com/seanpont/gobro"
	"io"
	"net"
	"strings"
	"sync"
)

// ===== SERVER ==============================================================

func simpleTapServer(args []string) {
	gobro.CheckArgs(args, 1, "Usage: tcptap SimpleTap <port>")
	NewSimpleServer().listen(args[0])
}

type SimpleTapServer struct {
	conns []net.Conn
	sync.Mutex
}

func NewSimpleServer() *SimpleTapServer {
	return &SimpleTapServer{
		conns: make([]net.Conn, 0),
	}
}

func (s *SimpleTapServer) listen(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	gobro.CheckErr(err)
	fmt.Println("SimpleTapServer listening on port", port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			gobro.LogErr(err)
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *SimpleTapServer) handleConn(conn net.Conn) {
	fmt.Println("handleConn:", conn.RemoteAddr())
	name := "Unknown"
	buff := make([]byte, 512)
	defer func() {
		fmt.Println("Close connection: ", conn.RemoteAddr())
		conn.Close()
		s.removeConn(conn)
		s.tap(name + " has left the party")
	}()

	// First message is used to identify the user
	n, err := conn.Read(buff[0:])
	if err != nil {
		if err != io.EOF {
			gobro.LogErr(err)
		}
		return
	}
	name = strings.Trim(string(buff[:n]), " \t\n\r")
	fmt.Println("read name:", name)
	s.addConn(conn)
	s.tap(name + " has joined the party")

	// All subsequent message get sent to everyone
	for {
		n, err := conn.Read(buff[0:])
		if err != nil {
			if err != io.EOF {
				gobro.LogErr(err)
			}
			return
		}
		message := name + ": " + strings.Trim(string(buff[:n]), " \t\n\r")
		s.tap(message)
	}
}

func (s *SimpleTapServer) addConn(conn net.Conn) {
	fmt.Println("addConn:", conn.RemoteAddr())
	s.Lock()
	defer s.Unlock()
	s.conns = append(s.conns, conn)
}

func (s *SimpleTapServer) removeConn(conn net.Conn) {
	fmt.Println("removeConn:", conn.RemoteAddr())
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

func (s *SimpleTapServer) tap(message string) {
	fmt.Println("tap:", message)
	message += "\n"
	s.Lock()
	defer s.Unlock()
	b := []byte(message)
	for _, conn := range s.conns {
		conn.Write(b)
	}
}
