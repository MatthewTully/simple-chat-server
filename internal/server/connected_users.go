package server

import (
	"bytes"
	"encoding/binary"
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

type UserInfo struct {
	Username   string
	UserColour string
}

type ConnectedUser struct {
	conn           net.Conn
	userInfo       UserInfo
	multiMessages  map[int]encoding.Protocol
	processChannel chan []byte
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
				if len(p) < encoding.HeaderSize {
					s.cfg.Logger.Printf("Server: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				s.cfg.Logger.Printf("Server: decode header\n")
				packetNum := binary.BigEndian.Uint16(p[0:])
				numPackets := binary.BigEndian.Uint16(p[2:])
				packetLen := binary.BigEndian.Uint16(p[4:])

				if len(p) < int(packetLen) {
					s.cfg.Logger.Printf("Server: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				packet := p[encoding.HeaderSize : packetLen+encoding.HeaderSize]

				if (packetLen + encoding.HeaderSize) > uint16(nr) {
					s.cfg.Logger.Printf("Server: add remaining bytes to overflow for next read: %v\n", p[packetLen+encoding.HeaderSize:])
					overFlow = p[packetLen+encoding.HeaderSize:]
					overFlow = append(overFlow, p[packetLen+encoding.HeaderSize:]...)
				}

				buffer := bytes.NewBuffer(packet)
				dataPacket := encoding.DecodePacket(buffer)
				if numPackets == 1 {
					s.ActionMessageType(dataPacket)
				} else {
					cu.multiMessages[int(packetNum)] = dataPacket
					if len(cu.multiMessages) == int(numPackets) {
						newProtocol := encoding.Protocol{}
						mergedData := []byte{}
						for i := 1; i <= int(numPackets); i++ {
							msg := cu.multiMessages[i]
							if i == 1 {
								newProtocol.MessageType = msg.MessageType
								newProtocol.Username = msg.Username
								newProtocol.UsernameSize = msg.UsernameSize
								newProtocol.UserColour = msg.UserColour
								newProtocol.DateTime = msg.DateTime
							}
							mergedData = append(mergedData, msg.Data[:msg.MsgSize]...)
						}
						cu.multiMessages = make(map[int]encoding.Protocol)
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
