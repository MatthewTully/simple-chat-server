package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

type UserInfo struct {
	Username   string
	UserColour string
}

type ConnectedUser struct {
	conn     net.Conn
	userInfo UserInfo
}

type serverConfig struct {
	ServerName string
	Logger     *log.Logger
}

type Server struct {
	cfg                *serverConfig
	LiveConns          map[string]ConnectedUser
	Listener           net.Listener
	MsgHistory         [][]byte
	MaxMsgHistorySize  uint
	MaxConnectionLimit uint
	Blacklist          []string
	rwmu               *sync.RWMutex
}

func NewServer(port string, historySize uint, logger *log.Logger) (Server, error) {
	l, err := NewListener(port)
	if err != nil {
		return Server{}, err
	}

	srvCfg := serverConfig{
		ServerName: "Chat Server",
		Logger:     logger,
	}

	srv := Server{
		LiveConns:         make(map[string]ConnectedUser),
		Listener:          l,
		cfg:               &srvCfg,
		MsgHistory:        [][]byte{},
		MaxMsgHistorySize: historySize,
		rwmu:              &sync.RWMutex{},
	}
	return srv, nil
}

func (s *Server) AddToLiveConns(userKey string, conn ConnectedUser) error {
	s.cfg.Logger.Printf("New connection - %s\n", userKey)
	s.rwmu.Lock()
	defer s.rwmu.Unlock()

	if uint(len(s.LiveConns)) >= s.MaxConnectionLimit {
		return fmt.Errorf("could not connect. Connection limit reached")
	}
	s.LiveConns[userKey] = conn
	return nil
}

func (s *Server) NewConnection(conn net.Conn, userInfo encoding.Protocol) (ConnectedUser, error) {
	newUser := ConnectedUser{
		conn: conn,
		userInfo: struct {
			Username   string
			UserColour string
		}{
			Username:   string(userInfo.Username[:userInfo.UsernameSize]),
			UserColour: string(userInfo.UserColour[:userInfo.UserColourSize]),
		},
	}
	err := s.AddToLiveConns(newUser.userInfo.Username, newUser)
	if err != nil {
		return ConnectedUser{}, err
	}
	time.Sleep(time.Millisecond)
	s.BroadcastActiveUsers()
	time.Sleep(time.Millisecond)
	err = s.SendHistory(newUser)
	if err != nil {
		s.cfg.Logger.Printf("Could not send history to new user (%v): %v", newUser.userInfo.Username, err)
	}
	time.Sleep(time.Millisecond)
	err = s.SentMessageToClient(newUser.userInfo.Username, []byte("Welcome to the server!\n"))
	time.Sleep(time.Millisecond)
	s.ProcessGroupMessage(s.cfg.ServerName, []byte(fmt.Sprintf("User %v has joined the server!\n", newUser.userInfo.Username)))
	if err != nil {
		s.cfg.Logger.Println(err.Error())
	}
	return newUser, nil
}

func (s *Server) DenyConnection(conn net.Conn, errMsg string) {
	errByte := []byte(errMsg)
	_, err := conn.Write(errByte)
	if err != nil {
		s.cfg.Logger.Println(err)
	}
	conn.Close()
}

func (s *Server) CloseConnection(user ConnectedUser) {
	s.ProcessGroupMessage(s.cfg.ServerName, []byte(fmt.Sprintf("User %v has left the server!\n", user.userInfo.Username)))
	s.rwmu.Lock()
	delete(s.LiveConns, user.userInfo.Username)
	s.rwmu.Unlock()
	user.conn.Close()
}

func (s *Server) CloseConnectionForUser(username string) {
	user, isActiveUser := s.IsActiveUser(username)
	if !isActiveUser {
		return
	}
	s.CloseConnection(user)
}

func (s *Server) IsActiveUser(username string) (ConnectedUser, bool) {
	s.rwmu.RLock()
	defer s.rwmu.RUnlock()
	user, exists := s.LiveConns[username]
	return user, exists
}

func (s *Server) SendHistory(user ConnectedUser) error {
	if len(s.MsgHistory) > 0 {
		s.rwmu.RLock()
		defer s.rwmu.RUnlock()
		for _, msg := range s.MsgHistory {
			err := s.SentMessageToClient(user.userInfo.Username, msg)
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
	if len(s.MsgHistory) >= int(s.MaxMsgHistorySize) {
		s.MsgHistory = s.MsgHistory[1:]

	}
	s.MsgHistory = append(s.MsgHistory, msg)
}

func (s *Server) AwaitHandshake(conn net.Conn) (encoding.Protocol, error) {
	buf := make([]byte, encoding.MaxPacketSize)
	for {
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				s.cfg.Logger.Printf("error reading from conn: %v\n", err)
			}
			return encoding.Protocol{}, err
		}
		if nr == 0 {
			continue
		}
		data := buf[0:nr]

		sd := bytes.Split(data, encoding.HeaderPattern[:])
		packetLen := binary.BigEndian.Uint16(sd[1][4:])
		packet := sd[1][encoding.HeaderSize : packetLen+encoding.HeaderSize]
		buffer := bytes.NewBuffer(packet)
		dataPacket := encoding.DecodePacket(buffer)
		if dataPacket.MessageType == encoding.RequestConnect {
			//TODO at this point can send encryption stuff and any other new connection data to client
			return dataPacket, nil
		}
	}
}

func (s *Server) GetAllActiveUsers() []UserInfo {
	s.rwmu.RLock()
	defer s.rwmu.RUnlock()

	activeUsers := []UserInfo{}
	for key := range s.LiveConns {
		activeUsers = append(activeUsers, s.LiveConns[key].userInfo)
	}
	return activeUsers

}

func (s *Server) BroadcastActiveUsers() {
	activeUsrSlice := []byte{}
	for _, data := range s.GetAllActiveUsers() {
		usrByteSlice := []byte(fmt.Sprintf("[%s]%v[white];", data.UserColour, data.Username))
		activeUsrSlice = append(activeUsrSlice, usrByteSlice...)
	}
	if len(activeUsrSlice) == 0 {
		return
	}

	toSend := encoding.PrepBytesForSending(activeUsrSlice, encoding.ServerActiveUsers, s.cfg.ServerName, "white")
	s.cfg.Logger.Printf("Total active users is: %v\n", len(s.GetAllActiveUsers()))
	s.cfg.Logger.Printf("BroadcastActiveUsers: len %v\n", len(toSend))
	s.BroadcastMessage(s.cfg.ServerName, toSend)
}
