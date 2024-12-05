package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

func (c *Client) SendHandshake(conn net.Conn) error {
	pubKeyBytes, err := crypto.RSAPublicKeyToBytes(c.cfg.RSAKeyPair.PublicKey)
	if err != nil {
		c.cfg.Logger.Printf("Client: %v", err)
		return err
	}
	handshake, err := encoding.PrepHandshakeForSending(pubKeyBytes, c.cfg.Username, c.cfg.UserColour)
	if err != nil {
		return fmt.Errorf("error creating packet to send: %v", err)
	}
	c.cfg.Logger.Printf("SendHandshake: len %v\n", len(handshake))
	_, err = conn.Write(handshake)
	if err != nil {
		c.cfg.Logger.Printf("Client: failed to send to server %s: %v\n", conn.RemoteAddr().String(), err)
		return fmt.Errorf("failed to send to server %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (c *Client) SendAESKey(conn net.Conn) error {
	packet, err := encoding.PrepAESForSending(c.cfg.ClientAESKey, c.ServerPubKey, c.cfg.RSAKeyPair)
	if err != nil {
		return fmt.Errorf("failed to prepare AES Packet to send to server %s: %v", c.ActiveConn.RemoteAddr().String(), err)
	}
	c.cfg.Logger.Printf("SendAESKey: len %v\n", len(packet))
	c.cfg.Logger.Printf("SendAESKey: packet %v\n", packet)
	_, err = conn.Write(packet)
	if err != nil {
		c.cfg.Logger.Printf("Client: failed to send to server %s: %v\n", conn.RemoteAddr().String(), err)
		return fmt.Errorf("failed to send to server %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (c *Client) AwaitHandshakeResponse(conn net.Conn) (encoding.MsgProtocol, error) {
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				c.cfg.Logger.Printf("error reading from conn: %v\n", err)
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
			c.cfg.Logger.Print("Client: handshake complete")
			return dataPacket, nil
		}
	}
}

func (c *Client) AwaitServerKey(conn net.Conn) ([]byte, error) {
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				c.cfg.Logger.Printf("error reading from conn: %v\n", err)
			}
			return nil, err
		}
		if nr == 0 {
			continue
		}
		data := buf[0:nr]
		c.cfg.Logger.Printf("Client: nr=%v\n", nr)
		c.cfg.Logger.Printf("Client: data=%v\n", data)
		sd := bytes.Split(data, encoding.HeaderPattern[:])
		encPacket := sd[1]
		packetLen := binary.BigEndian.Uint16(encPacket[4:])

		payload := encPacket[encoding.HeaderSize : packetLen+encoding.HeaderSize]
		buffer := bytes.NewBuffer(payload)
		dataPacket := encoding.DecodeAESPacket(buffer)

		decPayload, err := crypto.RSADecrypt(dataPacket.Data[:dataPacket.MsgSize], c.cfg.RSAKeyPair.PrivateKey)
		if err != nil {
			c.cfg.Logger.Printf("Client: error Decrypting payload: %v", err)
			return nil, err
		}
		err = crypto.RSAVerify(decPayload, dataPacket.Sig[:dataPacket.SigSize], c.ServerPubKey)
		if err != nil {
			c.cfg.Logger.Printf("Client: error verifying payload: %v", err)
			return nil, err
		}

		if dataPacket.MessageType == encoding.SendAESKey {
			c.cfg.Logger.Print("Client: AES Key Received complete")
			return decPayload, nil
		}
	}
}
