package server

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"net"

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
	publicKey      *rsa.PublicKey
	AESKey         []byte
}

func (cu *ConnectedUser) ProcessMessage(s *Server) {
	cu.processChannel = make(chan []byte)
	overFlow := []byte{}
	var data []byte
	go s.AwaitMessage(*cu)
	for {
		buf := <-cu.processChannel
		nr := len(buf)
		s.cfg.Logger.Printf("Server: in chan, Buf read = %v\n", buf)
		if len(overFlow) > 0 {
			s.cfg.Logger.Printf("Server: Using overflow\n")
			data = append(overFlow, buf[0:nr]...)
			overFlow = []byte{}
			s.cfg.Logger.Printf("Server: data with overflow and buf = %v\n", data)
		} else {
			data = buf[0:nr]
		}

		sd := bytes.Split(data, encoding.HeaderPattern[:])

		for i, p := range sd {
			s.cfg.Logger.Printf("%v: %v\n", i, p)
			switch {
			case i == 0 && len(p) > 0:
				s.cfg.Logger.Println("Server: tail end of the previous message. add to overflow")
				overFlow = append(overFlow, p...)
				continue
			case i == 1:
				if len(p) < encoding.AESEncryptHeaderSize {
					s.cfg.Logger.Printf("Server: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				s.cfg.Logger.Printf("Server: decode header\n")

				payloadSize := binary.BigEndian.Uint16(p[0:])

				if len(p) < int(payloadSize) {
					s.cfg.Logger.Printf("Server: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				payload := p[encoding.AESEncryptHeaderSize : payloadSize+encoding.AESEncryptHeaderSize]

				if (payloadSize + encoding.AESEncryptHeaderSize) > uint16(nr) {
					s.cfg.Logger.Printf("Server: add remaining bytes to overflow for next read: %v\n", p[payloadSize+encoding.AESEncryptHeaderSize:])
					overFlow = p[payloadSize+encoding.AESEncryptHeaderSize:]
					overFlow = append(overFlow, p[payloadSize+encoding.AESEncryptHeaderSize:]...)
				}

				decPayload, err := crypto.AESDecrypt(payload, cu.AESKey)
				if err != nil {
					s.cfg.Logger.Printf("Server: error Decrypting payload: %v", err)
					continue
				}

				packetNum := binary.BigEndian.Uint16(decPayload[0:])
				numPackets := binary.BigEndian.Uint16(decPayload[2:])
				packetLen := binary.BigEndian.Uint16(decPayload[4:])

				if len(decPayload) < int(packetLen) {
					s.cfg.Logger.Printf("Server: error, decrypted payload is not the full message")
					continue
				}

				packet := decPayload[encoding.HeaderSize : packetLen+encoding.HeaderSize]

				buffer := bytes.NewBuffer(packet)
				dataPacket := encoding.DecodeMsgPacket(buffer)
				if numPackets == 1 {
					s.ActionMessageType(dataPacket)
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
						s.ActionMessageTypeMultiMessage(newProtocol, mergedData)
					}
				}
				continue
			case i > 1:
				s.cfg.Logger.Printf("Server: add extra bytes to overflow for next read: %v\n", p[:])
				overFlow = append(overFlow, encoding.HeaderPattern[:]...)
				overFlow = append(overFlow, p...)
				continue
			default:
				continue
			}

		}

	}
}
