package server

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

type serverConfig struct {
	ServerName string
	Logger     *log.Logger
	RSAKeyPair crypto.RSAKeys
	AESKey     []byte
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

	priv, pub, err := crypto.GenerateRSAKeyPair()
	if err != nil {
		logger.Fatalf("could not generate RSA key pair for server: %v", err)
	}

	aesKey, err := crypto.GenerateAESSecretKey()
	if err != nil {
		logger.Fatalf("could not generate AES key for server: %v", err)
	}

	srvCfg := serverConfig{
		ServerName: "Chat Server",
		Logger:     logger,
		RSAKeyPair: crypto.RSAKeys{
			PrivateKey: priv,
			PublicKey:  pub,
		},
		AESKey: aesKey,
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
	_, exists := s.LiveConns[userKey]
	if exists {
		return fmt.Errorf("a user with same username is already connected")
	}

	s.LiveConns[userKey] = conn
	return nil
}

func (s *Server) NewConnection(newUser ConnectedUser) (ConnectedUser, error) {
	err := s.AddToLiveConns(newUser.userInfo.Username, newUser)
	if err != nil {
		return ConnectedUser{}, err
	}
	s.BroadcastActiveUsers()
	err = s.SendHistory(newUser)
	if err != nil {
		s.cfg.Logger.Printf("Could not send history to new user (%v): %v", newUser.userInfo.Username, err)
	}
	err = s.SentMessageToClient(newUser.userInfo.Username, []byte("Welcome to the server!\n"))
	s.ProcessGroupMessage(s.cfg.ServerName, []byte(fmt.Sprintf("User %v has joined the server!\n", newUser.userInfo.Username)))
	if err != nil {
		s.cfg.Logger.Println(err.Error())
	}
	return newUser, nil
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

func (s *Server) AwaitHandshake(conn net.Conn) (encoding.MsgProtocol, error) {
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				s.cfg.Logger.Printf("error reading from conn: %v\n", err)
			}
			return encoding.MsgProtocol{}, err
		}
		if nr == 0 {
			continue
		}
		data := buf[0:nr]

		sd := bytes.Split(data, encoding.HeaderPattern[:])
		packetLen := binary.BigEndian.Uint16(sd[1][4:])
		packet := sd[1][encoding.HeaderSize : packetLen+encoding.HeaderSize]
		buffer := bytes.NewBuffer(packet)
		dataPacket := encoding.DecodeMsgPacket(buffer)
		if dataPacket.MessageType == encoding.RequestConnect {
			s.cfg.Logger.Print("Server: handshake received")
			return dataPacket, nil
		}
	}
}

func (s *Server) SendHandshakeResponse(conn net.Conn) error {
	pubKeyBytes, err := crypto.RSAPublicKeyToBytes(s.cfg.RSAKeyPair.PublicKey)
	if err != nil {
		s.cfg.Logger.Printf("Server: %v", err)
		return err
	}
	handshake, err := encoding.PrepHandshakeForSending(pubKeyBytes, s.cfg.ServerName, "white")
	if err != nil {
		return fmt.Errorf("error creating packet to send: %v", err)
	}
	s.cfg.Logger.Printf("SendHandshakeResponse: len %v\n", len(handshake))
	_, err = conn.Write(handshake)
	if err != nil {
		s.cfg.Logger.Printf("Server: failed to send to user %s: %v\n", conn.RemoteAddr().String(), err)
		return fmt.Errorf("failed to send to user %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (s *Server) SendAESKey(conn net.Conn, cliPubKey *rsa.PublicKey) error {
	packet, err := encoding.PrepAESForSending(s.cfg.AESKey, cliPubKey, s.cfg.RSAKeyPair)
	if err != nil {
		return fmt.Errorf("failed to prepare AES Packet to send to user %s: %v", conn.RemoteAddr().String(), err)
	}
	s.cfg.Logger.Printf("SendAESKey: len %v\n", len(packet))
	s.cfg.Logger.Printf("SendAESKey: packet %v\n", packet)
	_, err = conn.Write(packet)
	if err != nil {
		s.cfg.Logger.Printf("Server: failed to send to user %s: %v\n", conn.RemoteAddr().String(), err)
		return fmt.Errorf("failed to send to user %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (s *Server) AwaitClientAESKey(conn net.Conn, cliPubKey *rsa.PublicKey) ([]byte, error) {
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				s.cfg.Logger.Printf("error reading from conn: %v\n", err)
			}
			return nil, err
		}
		if nr == 0 {
			continue
		}
		data := buf[0:nr]

		sd := bytes.Split(data, encoding.HeaderPattern[:])
		encPacket := sd[1]
		packetLen := binary.BigEndian.Uint16(encPacket[4:])

		payload := encPacket[encoding.HeaderSize : packetLen+encoding.HeaderSize]
		buffer := bytes.NewBuffer(payload)
		dataPacket := encoding.DecodeAESPacket(buffer)

		decPayload, err := crypto.RSADecrypt(dataPacket.Data[:dataPacket.MsgSize], s.cfg.RSAKeyPair.PrivateKey)
		if err != nil {
			s.cfg.Logger.Printf("Server: error Decrypting payload: %v", err)
			return nil, err
		}
		err = crypto.RSAVerify(decPayload, dataPacket.Sig[:dataPacket.SigSize], cliPubKey)
		if err != nil {
			s.cfg.Logger.Printf("Server: error verifying payload: %v", err)
			return nil, err
		}

		if dataPacket.MessageType == encoding.SendAESKey {
			s.cfg.Logger.Print("Server: AES Key Received complete")
			return decPayload, nil
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

	toSend, err := encoding.PrepBytesForSending(activeUsrSlice, encoding.ServerActiveUsers, s.cfg.ServerName, "white", s.cfg.AESKey)
	if err != nil {
		s.cfg.Logger.Printf("error creating packet to send: %v", err)
	}
	s.cfg.Logger.Printf("Total active users is: %v\n", len(s.GetAllActiveUsers()))
	s.cfg.Logger.Printf("BroadcastActiveUsers: len %v\n", len(toSend))
	s.BroadcastMessage(s.cfg.ServerName, toSend)
}
