package client

import (
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
	cfg         *ClientConfig
	ActiveConn  net.Conn
	LastCommand string
	TUI         *tview.Application
	chatView    *tview.TextView
}

func NewClient(cfg *ClientConfig) Client {
	return Client{
		cfg: cfg,
	}
}

func (c *Client) Connect(srvAddr string) error {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		//fmt.Printf("Could not connect to %v: %v\n", srvAddr, err)
		return err
	}

	c.ActiveConn = conn
	return nil
}

func (c *Client) AwaitMessage() {
	buf := make([]byte, encoding.MaxMessageSize)
	for {
		conn := c.ActiveConn
		if conn == nil {
			continue
		}

		nr, err := conn.Read(buf)
		if err != nil {
			if !strings.Contains(err.Error(), "closed network connection") {
				//fmt.Printf("error reading from conn: %v\n", err)
			}
			return
		}
		if nr == 0 {
			return
		}

		data := buf[0:nr]
		c.chatView.Write(data)
	}
}

func (c *Client) SendMessageToServer(msg []byte) error {
	_, err := c.ActiveConn.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to sent to server %s: %v", c.ActiveConn.RemoteAddr().String(), err)
	}
	return nil
}

func (c *Client) PushMessageToChatView(msg string) {
	msg = fmt.Sprintf("[%s]%s ~ [white]%s", c.cfg.UserColour, c.cfg.Username, msg)
	c.chatView.Write([]byte(msg))
}
