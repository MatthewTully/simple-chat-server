package server

import (
	"fmt"
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

func (s *Server) AddToLiveConns(userKey string, conn *ConnectedUser) error {
	s.cfg.Logger.Printf("New connection - %s\n", userKey)
	s.rwmu.Lock()
	defer s.rwmu.Unlock()

	if uint(len(s.LiveConns)) >= s.MaxConnectionLimit {
		return fmt.Errorf("could not connect. Connection limit reached")
	}
	_, exists := s.LiveConns[userKey]
	if exists {
		return fmt.Errorf("a user with same username is already connected")
	}

	s.LiveConns[userKey] = conn
	return nil
}

func (s *Server) NewConnection(newUser ConnectedUser) (*ConnectedUser, error) {
	err := s.AddToLiveConns(newUser.userInfo.Username, &newUser)
	if err != nil {
		return &ConnectedUser{}, err
	}
	s.BroadcastActiveUsers()
	err = s.SendHistory(&newUser)
	if err != nil {
		s.cfg.Logger.Printf("Could not send history to new user (%v): %v", newUser.userInfo.Username, err)
	}
	err = s.SentMessageToClient(newUser.userInfo.Username, []byte("Welcome to the server!\n"))
	s.ProcessGroupMessage(s.cfg.ServerName, []byte(fmt.Sprintf("User %v has joined the server!\n", newUser.userInfo.Username)))
	if err != nil {
		s.cfg.Logger.Println(err.Error())
	}
	return &newUser, nil
}

func (s *Server) DenyConnection(conn net.Conn, errMsg string) {
	errByte := []byte(errMsg)
	toSend, err := encoding.PrepBytesForSending(errByte, encoding.ErrorMessage, s.cfg.ServerName, "white", s.cfg.AESKey)
	if err != nil {
		s.cfg.Logger.Printf("error creating packet to send: %v", err)
	}
	_, err = conn.Write(toSend)
	if err != nil {
		s.cfg.Logger.Println(err)
	}
	conn.Close()
}

func (s *Server) CloseConnection(user *ConnectedUser) {
	s.rwmu.Lock()
	delete(s.LiveConns, user.userInfo.Username)
	s.rwmu.Unlock()
	s.SendDisconnectionNotification(user)
	user.conn.Close()
	s.cfg.Logger.Printf("Connection closed for user %v", user.userInfo.Username)
	s.ProcessGroupMessage(s.cfg.ServerName, []byte(fmt.Sprintf("User %v has left the server!\n", user.userInfo.Username)))
	s.BroadcastActiveUsers()
}

func (s *Server) CloseConnectionForUser(username string) {
	user, isActiveUser := s.IsActiveUser(username)
	if !isActiveUser {
		return
	}
	s.CloseConnection(user)
}
