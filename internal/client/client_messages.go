package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

func (c *Client) ActionMessageType(p encoding.MsgProtocol) {
	switch p.MessageType {
	case encoding.Message:
		c.cfg.Logger.Printf("Message type received: Message\n")
		c.chatView.Write(p.Data[:p.MsgSize])
	case encoding.ErrorMessage:
		c.cfg.Logger.Printf("Message type received: Error Message\n")
		msg := []byte("[red]Error: ")
		msg = append(msg, p.Data[:p.MsgSize]...)
		msg = append(msg, []byte("[white]")...)
		c.chatView.Write(msg)
	case encoding.ServerActiveUsers:
		c.cfg.Logger.Printf("Message type received: Active Users\n")
		c.activeUsersView.Clear()
		activeUsr := strings.Split(string(p.Data[:p.MsgSize]), ";")
		for _, usr := range activeUsr {
			c.activeUsersView.Write([]byte(usr + "\n"))
		}
	case encoding.RequestDisconnect:
		c.cfg.Logger.Printf("Message type received: Request Disconnect\n")
		c.ActiveConn.Close()
		c.chatView.Clear()
		c.PushToChatView("You have been disconnected.")
		c.activeUsersView.Clear()
		c.showHomePage()
	}
}

func (c *Client) ActionMessageTypeMultiMessage(p encoding.MsgProtocol, data []byte) {
	switch p.MessageType {
	case encoding.Message:
		c.cfg.Logger.Printf("Message type received: Message\n")
		c.chatView.Write(data)
	case encoding.ErrorMessage:
		c.cfg.Logger.Printf("Message type received: Error Message\n")
		msg := []byte("[red]Error: ")
		msg = append(msg, data...)
		msg = append(msg, []byte("[white]")...)
		c.chatView.Write(msg)
	case encoding.ServerActiveUsers:
		c.cfg.Logger.Printf("Message type received: Active Users\n")
		c.activeUsersView.Clear()
		activeUsr := strings.Split(string(data), ";")
		for _, usr := range activeUsr {
			c.activeUsersView.Write([]byte(usr + "\n"))
		}
	}
}

func (c *Client) ProcessMessage() {
	c.processChannel = make(chan []byte)
	overFlow := []byte{}
	var data []byte
	go c.AwaitMessage()
	for {
		buf := <-c.processChannel
		nr := len(buf)
		c.cfg.Logger.Printf("in chan, Buf read = %v\n", buf)
		if len(overFlow) > 0 {
			c.cfg.Logger.Printf("Using overflow\n")
			data = append(overFlow, buf[0:nr]...)
			overFlow = []byte{}
			c.cfg.Logger.Printf("data with overflow and buf = %v\n", data)
		} else {
			data = buf[0:nr]
		}

		sd := bytes.Split(data, encoding.HeaderPattern[:])

		for i, p := range sd {
			c.cfg.Logger.Printf("%v: %v\n", i, p)
			switch {
			case i == 0 && len(p) > 0:
				c.cfg.Logger.Println("tail end of the previous message. add to overflow")
				overFlow = append(overFlow, p...)
				continue
			case i == 1:
				if len(p) < encoding.AESEncryptHeaderSize {
					c.cfg.Logger.Printf("packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				c.cfg.Logger.Printf("decode header\n")

				payloadSize := binary.BigEndian.Uint16(p[0:])

				if len(p) < int(payloadSize) {
					c.cfg.Logger.Printf("packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				payload := p[encoding.AESEncryptHeaderSize : payloadSize+encoding.AESEncryptHeaderSize]

				if (payloadSize + encoding.AESEncryptHeaderSize) > uint16(nr) {
					c.cfg.Logger.Printf("add remaining bytes to overflow for next read: %v\n", p[payloadSize+encoding.AESEncryptHeaderSize:])
					overFlow = p[payloadSize+encoding.AESEncryptHeaderSize:]
					overFlow = append(overFlow, p[payloadSize+encoding.AESEncryptHeaderSize:]...)
				}

				decPayload, err := crypto.AESDecrypt(payload, c.ServerAESKey)
				if err != nil {
					c.cfg.Logger.Printf("error Decrypting payload: %v", err)
					continue
				}

				packetNum := binary.BigEndian.Uint16(decPayload[0:])
				numPackets := binary.BigEndian.Uint16(decPayload[2:])
				packetLen := binary.BigEndian.Uint16(decPayload[4:])

				if len(decPayload) < int(packetLen) {
					c.cfg.Logger.Printf("error, decrypted payload is not the full message")
					continue
				}
				packet := decPayload[encoding.HeaderSize : packetLen+encoding.HeaderSize]

				buffer := bytes.NewBuffer(packet)
				dataPacket := encoding.DecodeMsgPacket(buffer)
				if numPackets == 1 {
					c.ActionMessageType(dataPacket)
				} else {
					c.multiMessages[int(packetNum)] = dataPacket
					if len(c.multiMessages) == int(numPackets) {
						newProtocol := encoding.MsgProtocol{}
						mergedData := []byte{}
						for i := 1; i <= int(numPackets); i++ {
							msg := c.multiMessages[i]
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
						c.multiMessages = make(map[int]encoding.MsgProtocol)
						c.ActionMessageTypeMultiMessage(newProtocol, mergedData)
					}
				}
				continue
			case i > 1:
				c.cfg.Logger.Printf("add extra bytes to overflow for next read: %v\n", p[:])
				overFlow = append(overFlow, encoding.HeaderPattern[:]...)
				overFlow = append(overFlow, p...)
				continue
			default:
				continue
			}

		}

	}
}

func (c *Client) AwaitMessage() {
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		var data []byte
		conn := c.ActiveConn
		if conn == nil {
			c.chatView.Clear()
			c.PushToChatView("Connection has been lost, or no active connection.")
			c.activeUsersView.Clear()
			c.showHomePage()
			return
		}

		c.cfg.Logger.Printf("Buff before read=%v\n", buf)
		nr, err := conn.Read(buf[:])
		c.cfg.Logger.Printf("nr=%v\n", nr)
		data = buf[0:nr]
		if nr == 0 {
			conn.Close()
			return
		}

		if err != nil {
			if !strings.Contains(err.Error(), "closed network connection") {
				c.cfg.Logger.Printf("error reading from conn: %v\n", err)
			}
			c.PushToChatView("Connection has been lost, please try to reconnect.")
			c.showHomePage()
			conn.Close()
			return
		}
		c.cfg.Logger.Printf("data read from conn=%v\n", data)
		c.processChannel <- data

	}
}

func (c *Client) SendMessageToServer(msg []byte) error {
	toSend, err := encoding.PrepBytesForSending(msg, encoding.Message, c.cfg.Username, c.cfg.UserColour, c.cfg.ClientAESKey)
	if err != nil {
		return fmt.Errorf("error creating packet to send: %v", err)
	}
	c.cfg.Logger.Printf("SendMessageToServer: len %v\n", len(toSend))
	_, err = c.ActiveConn.Write(toSend)
	if err != nil {
		return fmt.Errorf("failed to send to server %s: %v", c.ActiveConn.RemoteAddr().String(), err)
	}
	return nil
}

func (c *Client) SendWhisperToServer(msg []byte) error {
	toSend, err := encoding.PrepBytesForSending(msg, encoding.WhisperMessage, c.cfg.Username, c.cfg.UserColour, c.cfg.ClientAESKey)
	if err != nil {
		return fmt.Errorf("error creating packet to send: %v", err)
	}
	c.cfg.Logger.Printf("SendWhisperToServer: len %v\n", len(toSend))
	_, err = c.ActiveConn.Write(toSend)
	if err != nil {
		return fmt.Errorf("failed to send to server %s: %v", c.ActiveConn.RemoteAddr().String(), err)
	}
	return nil
}

func (c *Client) PushSentMessageToChatView(msg string) {
	dateTime := time.Now().UTC()
	msg = fmt.Sprintf("[white]%v[white] [%s]%s ~ [white]%s", dateTime.Format("02/01/06 15:04"), c.cfg.UserColour, c.cfg.Username, msg)
	c.chatView.Write([]byte(msg))
}
