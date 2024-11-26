package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/MatthewTully/simple-chat-server/internal/encoding"
	"github.com/rivo/tview"
)

type ClientConfig struct {
	Username   string
	UserColour string
}

type Client struct {
	cfg             *ClientConfig
	ActiveConn      net.Conn
	processChannel  chan []byte
	LastCommand     string
	TUI             *tview.Application
	chatView        *tview.TextView
	activeUsersView *tview.TextView
}

func NewClient(cfg *ClientConfig) Client {
	return Client{
		cfg: cfg,
	}
}

func (c *Client) Connect(srvAddr string) error {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		fmt.Printf("Client: Could not connect to %v: %v\n", srvAddr, err)
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
		fmt.Printf("Message type received: Message")
		c.chatView.Write(p.Data[:p.MsgSize])
	case encoding.ServerActiveUsers:
		fmt.Printf("Message type received: Active Users")
		activeUsr := strings.Split(string(p.Data[:p.MsgSize]), ";")
		for _, data := range activeUsr {
			c.activeUsersView.Write([]byte(data + "\n"))
		}
	}

}

func (c *Client) SendHandshake(conn net.Conn) error {
	handshake := encoding.PrepBytesForSending([]byte{}, encoding.RequestConnect, c.cfg.Username, c.cfg.UserColour)
	fmt.Printf("SendHandshake: len %v\n", len(handshake))
	_, err := conn.Write(handshake)
	if err != nil {
		fmt.Printf("Client: failed to send to server %s: %v\n", conn.RemoteAddr().String(), err)
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
		if len(overFlow) > 0 {
			fmt.Printf("Client: Using overflow\n")
			data = append(overFlow, buf[0:nr]...)
			overFlow = []byte{}
		} else {
			data = buf[0:nr]
		}

		sd := bytes.Split(data, encoding.HeaderPattern[:])

		for i, packets := range sd {
			fmt.Printf("%v: %v\n", i, packets)
			switch {
			case i == 0 && len(packets) > 0:
				//overFlow = append(overFlow, encoding.HeaderPattern[:]...)
				fmt.Printf("Client: add to overflow\n")
				overFlow = append(overFlow, packets...)
				continue
			case i == 1:
				fmt.Printf("Client: decode header\n")
				packetNum := binary.BigEndian.Uint16(packets[0:])
				numPackets := binary.BigEndian.Uint16(packets[2:])
				packetLen := binary.BigEndian.Uint16(packets[4:])
				if len(packets) > int(packetLen) {
					packets = append(packets, overFlow...)
					fmt.Printf("new packets: %v", packets)
				}
				packet := packets[encoding.HeaderSize : packetLen+encoding.HeaderSize]

				if (packetLen + encoding.HeaderSize) > uint16(nr) {
					fmt.Printf("add remaining bytes to overflow for next read: %v\n", packets[packetLen+encoding.HeaderSize:])
					overFlow = packets[packetLen+encoding.HeaderSize:]
				}

				buffer := bytes.NewBuffer(packet)
				dataPacket := encoding.DecodePacket(buffer)
				if packetNum == numPackets {
					c.ActionMessageType(dataPacket)
				}
				continue
			case i > 1:
				fmt.Printf("add extra bytes to overflow for next read: %v\n", packets[:])
				overFlow = append(overFlow, encoding.HeaderPattern[:]...)
				overFlow = append(overFlow, packets...)
				continue
			default:
				continue
			}

		}

	}
}

func (c *Client) AwaitMessage() {
	buf := make([]byte, encoding.MaxPacketSize)
	var data []byte
	for {
		conn := c.ActiveConn
		if conn == nil {
			continue
		}

		nr, err := conn.Read(buf)
		if err != nil {
			if !strings.Contains(err.Error(), "closed network connection") {
				fmt.Printf("Client: error reading from conn: %v\n", err)
			}
			return
		}
		if nr == 0 {
			return
		}
		fmt.Printf("Client: nr=%v\n", nr)
		data = buf[0:nr]
		c.processChannel <- data

	}
}

func (c *Client) SendMessageToServer(msg []byte) error {
	toSend := encoding.PrepBytesForSending(msg, encoding.Message, c.cfg.Username, c.cfg.UserColour)
	fmt.Printf("SendMessageToServer: len %v\n", len(toSend))
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
