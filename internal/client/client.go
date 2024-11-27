package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/MatthewTully/simple-chat-server/internal/encoding"
	"github.com/rivo/tview"
)

type ClientConfig struct {
	Username   string `json:"username"`
	UserColour string `json:"user_colour"`
	Logger     *log.Logger
}

type Client struct {
	cfg             *ClientConfig
	ActiveConn      net.Conn
	processChannel  chan []byte
	LastCommand     string
	TUI             *tview.Application
	chatView        *tview.TextView
	activeUsersView *tview.TextView
	multiMessages   map[int]encoding.Protocol
}

func NewClient(cfg *ClientConfig) Client {
	return Client{
		cfg:           cfg,
		multiMessages: make(map[int]encoding.Protocol),
	}
}

func (c *Client) Connect(srvAddr string) error {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		c.cfg.Logger.Printf("Client: Could not connect to %v: %v\n", srvAddr, err)
		return err
	}

	err = c.SendHandshake(conn)
	if err != nil {
		conn.Close()
		return err
	}
	c.ActiveConn = conn
	return nil
}

func (c *Client) ActionMessageType(p encoding.Protocol) {
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
	}
}

func (c *Client) ActionMessageTypeMultiMessage(p encoding.Protocol, data []byte) {
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

func (c *Client) SendHandshake(conn net.Conn) error {
	handshake := encoding.PrepBytesForSending([]byte{}, encoding.RequestConnect, c.cfg.Username, c.cfg.UserColour)
	c.cfg.Logger.Printf("SendHandshake: len %v\n", len(handshake))
	_, err := conn.Write(handshake)
	if err != nil {
		c.cfg.Logger.Printf("Client: failed to send to server %s: %v\n", conn.RemoteAddr().String(), err)
		return fmt.Errorf("failed to send to server %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (c *Client) ProcessMessage() {
	c.processChannel = make(chan []byte)
	overFlow := []byte{}
	var data []byte
	go c.AwaitMessage()
	for {
		buf := <-c.processChannel
		nr := len(buf)
		c.cfg.Logger.Printf("Client: in chan, Buf read = %v\n", buf)
		if len(overFlow) > 0 {
			c.cfg.Logger.Printf("Client: Using overflow\n")
			data = append(overFlow, buf[0:nr]...)
			overFlow = []byte{}
			c.cfg.Logger.Printf("Client: data with overflow and buf = %v\n", data)
		} else {
			data = buf[0:nr]
		}

		sd := bytes.Split(data, encoding.HeaderPattern[:])

		for i, p := range sd {
			c.cfg.Logger.Printf("%v: %v\n", i, p)
			switch {
			case i == 0 && len(p) > 0:
				c.cfg.Logger.Println("Client: tail end of the previous message. add to overflow")
				overFlow = append(overFlow, p...)
				continue
			case i == 1:
				if len(p) < encoding.HeaderSize {
					c.cfg.Logger.Printf("Client: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				c.cfg.Logger.Printf("Client: decode header\n")
				packetNum := binary.BigEndian.Uint16(p[0:])
				numPackets := binary.BigEndian.Uint16(p[2:])
				packetLen := binary.BigEndian.Uint16(p[4:])

				if len(p) < int(packetLen) {
					c.cfg.Logger.Printf("Client: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				packet := p[encoding.HeaderSize : packetLen+encoding.HeaderSize]

				if (packetLen + encoding.HeaderSize) > uint16(nr) {
					c.cfg.Logger.Printf("Client: add remaining bytes to overflow for next read: %v\n", p[packetLen+encoding.HeaderSize:])
					overFlow = p[packetLen+encoding.HeaderSize:]
					overFlow = append(overFlow, p[packetLen+encoding.HeaderSize:]...)
				}

				buffer := bytes.NewBuffer(packet)
				dataPacket := encoding.DecodePacket(buffer)
				if numPackets == 1 {
					c.ActionMessageType(dataPacket)
				} else {
					c.multiMessages[int(packetNum)] = dataPacket
					if len(c.multiMessages) == int(numPackets) {
						newProtocol := encoding.Protocol{}
						mergedData := []byte{}
						for i := 1; i <= int(numPackets); i++ {
							msg := c.multiMessages[i]
							if i == 1 {
								newProtocol.MessageType = msg.MessageType
								newProtocol.Username = msg.Username
								newProtocol.UsernameSize = msg.UsernameSize
								newProtocol.UserColour = msg.UserColour
								newProtocol.DateTime = msg.DateTime
							}
							mergedData = append(mergedData, msg.Data[:msg.MsgSize]...)
						}
						c.multiMessages = make(map[int]encoding.Protocol)
						c.ActionMessageTypeMultiMessage(newProtocol, mergedData)
					}
				}
				continue
			case i > 1:
				c.cfg.Logger.Printf("Client: add extra bytes to overflow for next read: %v\n", p[:])
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
			continue
		}

		c.cfg.Logger.Printf("Client: Buff before read=%v\n", buf)
		nr, err := conn.Read(buf[:])
		c.cfg.Logger.Printf("Client: nr=%v\n", nr)
		data = buf[0:nr]
		if nr == 0 {
			return
		}

		if err != nil {
			if !strings.Contains(err.Error(), "closed network connection") {
				c.cfg.Logger.Printf("Client: error reading from conn: %v\n", err)
			}
			return
		}
		c.cfg.Logger.Printf("Client: data read from conn=%v\n", data)
		c.processChannel <- data

	}
}

func (c *Client) SendMessageToServer(msg []byte) error {
	toSend := encoding.PrepBytesForSending(msg, encoding.Message, c.cfg.Username, c.cfg.UserColour)
	c.cfg.Logger.Printf("SendMessageToServer: len %v\n", len(toSend))
	_, err := c.ActiveConn.Write(toSend)
	if err != nil {
		return fmt.Errorf("failed to send to server %s: %v", c.ActiveConn.RemoteAddr().String(), err)
	}
	return nil
}

func (c *Client) PushMessageToChatView(msg string) {
	msg = fmt.Sprintf("[%s]%s ~ [white]%s", c.cfg.UserColour, c.cfg.Username, msg)
	c.chatView.Write([]byte(msg))
}
