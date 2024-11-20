package server

import (
	"fmt"
	"net"
	"sync"
)

const (
	maxSize    = 512
	maxLogSize = 20
)

type Server struct {
	LiveConns  map[string]net.Conn
	Listener   net.Listener
	Protocol   Protocol
	MsgHistory [][]byte
	rwmu       *sync.RWMutex
}

type Protocol struct {
	MaxSize int
}

func NewServer(port string) (Server, error) {
	l, err := NewListener(port)
	if err != nil {
		return Server{}, err
	}

	srv := Server{
		LiveConns: make(map[string]net.Conn),
		Listener:  l,
		Protocol: Protocol{
			MaxSize: maxSize,
		},
		MsgHistory: [][]byte{},
		rwmu:       &sync.RWMutex{},
	}
	return srv, nil
}

func (s *Server) NewConnection(conn net.Conn) {
	user := conn.RemoteAddr().String()
	fmt.Printf("New connection - %s\n", user)

	s.rwmu.Lock()
	s.LiveConns[user] = conn
	s.rwmu.Unlock()

	err := s.SendHistory(conn)
	if err != nil {
		fmt.Printf("Could not send history to new user (%v): %v", user, err)
	}
	s.BroadcastMessage("", []byte(fmt.Sprintf("User %v has joined the server!\n", user)))
	err = s.SentMessageToClient(user, []byte("Welcome to the server!\n"))
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (s *Server) CloseConnection(conn net.Conn) {
	user := conn.RemoteAddr().String()
	s.BroadcastMessage("", []byte(fmt.Sprintf("User %v has left the server!\n", user)))
	s.rwmu.Lock()
	delete(s.LiveConns, user)
	s.rwmu.Unlock()
	conn.Close()
}

func (s *Server) SendHistory(conn net.Conn) error {
	if len(s.MsgHistory) > 0 {
		s.rwmu.RLock()
		defer s.rwmu.RUnlock()
		for _, msg := range s.MsgHistory {
			err := s.SentMessageToClient(conn.RemoteAddr().String(), msg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) AddMsgToHistory(msg []byte) {
	s.rwmu.Lock()
	defer s.rwmu.Unlock()
	if len(s.MsgHistory) >= maxLogSize {
		s.MsgHistory = s.MsgHistory[1:]

	}
	s.MsgHistory = append(s.MsgHistory, msg)
}
