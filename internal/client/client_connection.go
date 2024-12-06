package client

import (
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
	"github.com/MatthewTully/simple-chat-server/internal/server"
)

func (c *Client) Connect(srvAddr string) error {
	conn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		c.cfg.Logger.Printf("Could not connect to %v: %v\n", srvAddr, err)
		return err
	}

	err = c.SendHandshake(conn)
	if err != nil {
		conn.Close()
		return err
	}
	res, err := c.AwaitHandshakeResponse(conn)
	if err != nil {
		conn.Close()
		return err
	}
	key, err := crypto.BytesToRSAPublicKey(res.Data[:res.MsgSize])
	if err != nil {
		conn.Close()
		return err
	}
	c.ServerPubKey = key

	err = c.SendAESKey(conn)
	if err != nil {
		conn.Close()
		return err
	}

	aes, err := c.AwaitServerKey(conn)
	if err != nil {
		conn.Close()
		return err
	}
	c.ServerAESKey = aes
	c.ActiveConn = conn
	go c.ProcessMessage()
	return nil
}

func (c *Client) SendDisconnectionRequest() {
	toSend, err := encoding.PrepBytesForSending([]byte{}, encoding.RequestDisconnect, c.cfg.Username, c.cfg.UserColour, c.cfg.ClientAESKey)
	if err != nil {
		c.cfg.Logger.Printf("error creating packet to send: %v", err)
	}
	c.cfg.Logger.Printf("SendDisconnectionRequest: len %v\n", len(toSend))
	_, err = c.ActiveConn.Write(toSend)
	if err != nil {
		c.cfg.Logger.Printf("failed to send to server %s: %v", c.ActiveConn.RemoteAddr().String(), err)
	}
}

func (c *Client) SetAsHost(srv *server.Server) {
	c.Host = true
	c.HostServer = srv
}
