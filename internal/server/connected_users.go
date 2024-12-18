package server

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

type UserInfo struct {
	Username   string
	UserColour string
}

type ConnectedUser struct {
	conn           net.Conn
	userInfo       UserInfo
	multiMessages  map[int]encoding.MsgProtocol
	processChannel chan []byte
	keepAliveTimer *time.Timer
	publicKey      *rsa.PublicKey
	AESKey         []byte
}

func (cu *ConnectedUser) ProcessMessage(s *Server) {
	cu.processChannel = make(chan []byte)
	keepAlive := time.NewTimer(time.Second * 30)
	cu.keepAliveTimer = keepAlive
	overFlow := []byte{}
	var data []byte
	go s.AwaitMessage(cu)
	for {
		select {
		case <-keepAlive.C:
			s.cfg.Logger.Printf("timer triggered for user %v, sending disconnect.", cu.userInfo.Username)
			s.CloseConnectionForUser(cu.userInfo.Username)
		case buf := <-cu.processChannel:
			nr := len(buf)
			s.cfg.Logger.Printf("in chan, Buf read = %v\n", buf)
			if len(overFlow) > 0 {
				s.cfg.Logger.Printf("Using overflow\n")
				data = append(overFlow, buf[0:nr]...)
				overFlow = []byte{}
				s.cfg.Logger.Printf("data with overflow and buf = %v\n", data)
			} else {
				data = buf[0:nr]
			}

			sd := bytes.Split(data, encoding.HeaderPattern[:])

			for i, p := range sd {
				s.cfg.Logger.Printf("%v: %v\n", i, p)
				switch {
				case i == 0 && len(p) > 0:
					s.cfg.Logger.Println("tail end of the previous message. add to overflow")
					overFlow = append(overFlow, p...)
					continue
				case i == 1:
					if len(p) < encoding.AESEncryptHeaderSize {
						s.cfg.Logger.Printf("packet is not the full message. add to overflow\n")
						overFlow = append(overFlow, encoding.HeaderPattern[:]...)
						overFlow = append(overFlow, p...)
						continue
					}
					s.cfg.Logger.Printf("decode header\n")

					payloadSize := binary.BigEndian.Uint16(p[0:])

					if len(p) < int(payloadSize) {
						s.cfg.Logger.Printf("packet is not the full message. add to overflow\n")
						overFlow = append(overFlow, encoding.HeaderPattern[:]...)
						overFlow = append(overFlow, p...)
						continue
					}
					payload := p[encoding.AESEncryptHeaderSize : payloadSize+encoding.AESEncryptHeaderSize]

					if (payloadSize + encoding.AESEncryptHeaderSize) > uint16(nr) {
						s.cfg.Logger.Printf("add remaining bytes to overflow for next read: %v\n", p[payloadSize+encoding.AESEncryptHeaderSize:])
						overFlow = p[payloadSize+encoding.AESEncryptHeaderSize:]
						overFlow = append(overFlow, p[payloadSize+encoding.AESEncryptHeaderSize:]...)
					}

					decPayload, err := crypto.AESDecrypt(payload, cu.AESKey)
					if err != nil {
						s.cfg.Logger.Printf("error Decrypting payload: %v", err)
						continue
					}

					packetNum := binary.BigEndian.Uint16(decPayload[0:])
					numPackets := binary.BigEndian.Uint16(decPayload[2:])
					packetLen := binary.BigEndian.Uint16(decPayload[4:])

					if len(decPayload) < int(packetLen) {
						s.cfg.Logger.Printf("error, decrypted payload is not the full message")
						continue
					}

					packet := decPayload[encoding.HeaderSize : packetLen+encoding.HeaderSize]

					buffer := bytes.NewBuffer(packet)
					dataPacket := encoding.DecodeMsgPacket(buffer)
					if numPackets == 1 {
						s.ActionMessageType(dataPacket, dataPacket.Data[:dataPacket.MsgSize])
					} else {
						cu.multiMessages[int(packetNum)] = dataPacket
						if len(cu.multiMessages) == int(numPackets) {
							newProtocol := encoding.MsgProtocol{}
							mergedData := []byte{}
							for i := 1; i <= int(numPackets); i++ {
								msg := cu.multiMessages[i]
								if i == 1 {
									newProtocol.MessageType = msg.MessageType
									newProtocol.Username = msg.Username
									newProtocol.UsernameSize = msg.UsernameSize
									newProtocol.UserColour = msg.UserColour
									newProtocol.UserColourSize = msg.UserColourSize
									newProtocol.DateTime = msg.DateTime
								}
								mergedData = append(mergedData, msg.Data[:msg.MsgSize]...)
							}
							cu.multiMessages = make(map[int]encoding.MsgProtocol)
							s.ActionMessageType(newProtocol, mergedData)
						}
					}
					continue
				case i > 1:
					s.cfg.Logger.Printf("add extra bytes to overflow for next read: %v\n", p[:])
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				default:
					continue
				}

			}
		}
	}
}

func (s *Server) IsActiveUser(username string) (*ConnectedUser, bool) {
	s.rwmu.RLock()
	defer s.rwmu.RUnlock()
	user, exists := s.LiveConns[username]
	return user, exists
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
		if data.Username == s.cfg.HostUser {
			data.Username = data.Username + " (host)"
		}
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

func (s *Server) BanUser(username string) bool {
	user, exists := s.IsActiveUser(username)
	if exists {
		ip := strings.Split(user.conn.LocalAddr().String(), ":")[0]
		s.Blacklist = append(s.Blacklist, ip)
		s.CloseConnectionForUser(username)
		return true
	}
	return false
}

func (s *Server) ActionKeepAlive(username string) {
	user, exists := s.IsActiveUser(username)
	if !exists {
		return
	}
	s.cfg.Logger.Printf("Keep alive received for user %v, extending timer", username)
	user.keepAliveTimer.Reset(time.Second * 30)
}
