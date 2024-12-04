package client

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
	"github.com/rivo/tview"
)

type ClientConfig struct {
	Username     string `json:"username"`
	UserColour   string `json:"user_colour"`
	Logger       *log.Logger
	RSAKeyPair   crypto.RSAKeys
	ClientAESKey []byte
}

type Client struct {
	cfg             *ClientConfig
	ActiveConn      net.Conn
	ServerAESKey    []byte
	ServerPubKey    *rsa.PublicKey
	processChannel  chan []byte
	LastCommand     string
	TUI             *tview.Application
	chatView        *tview.TextView
	activeUsersView *tview.TextView
	multiMessages   map[int]encoding.MsgProtocol
	userCmdArg      string
	tuiPages        *tview.Pages
}

func NewClient(cfg *ClientConfig) Client {
	priv, pub, err := crypto.GenerateRSAKeyPair()
	if err != nil {
		cfg.Logger.Fatalf("could not generate key RSA pair for client: %v", err)
	}
	cfg.RSAKeyPair = crypto.RSAKeys{
		PrivateKey: priv,
		PublicKey:  pub,
	}

	aesKey, err := crypto.GenerateAESSecretKey()
	if err != nil {
		cfg.Logger.Fatalf("could not generate AES key for client: %v", err)
	}
	cfg.ClientAESKey = aesKey

	return Client{
		cfg:           cfg,
		multiMessages: make(map[int]encoding.MsgProtocol),
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
				if len(p) < encoding.AESEncryptHeaderSize {
					c.cfg.Logger.Printf("Client: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				c.cfg.Logger.Printf("Client: decode header\n")

				payloadSize := binary.BigEndian.Uint16(p[0:])

				if len(p) < int(payloadSize) {
					c.cfg.Logger.Printf("Client: packet is not the full message. add to overflow\n")
					overFlow = append(overFlow, encoding.HeaderPattern[:]...)
					overFlow = append(overFlow, p...)
					continue
				}
				payload := p[encoding.AESEncryptHeaderSize : payloadSize+encoding.AESEncryptHeaderSize]

				if (payloadSize + encoding.AESEncryptHeaderSize) > uint16(nr) {
					c.cfg.Logger.Printf("Client: add remaining bytes to overflow for next read: %v\n", p[payloadSize+encoding.AESEncryptHeaderSize:])
					overFlow = p[payloadSize+encoding.AESEncryptHeaderSize:]
					overFlow = append(overFlow, p[payloadSize+encoding.AESEncryptHeaderSize:]...)
				}

				decPayload, err := crypto.AESDecrypt(payload, c.ServerAESKey)
				if err != nil {
					c.cfg.Logger.Printf("Client: error Decrypting payload: %v", err)
					continue
				}

				packetNum := binary.BigEndian.Uint16(decPayload[0:])
				numPackets := binary.BigEndian.Uint16(decPayload[2:])
				packetLen := binary.BigEndian.Uint16(decPayload[4:])

				if len(decPayload) < int(packetLen) {
					c.cfg.Logger.Printf("Client: error, decrypted payload is not the full message")
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

func (c *Client) PushMessageToChatView(msg string) {
	msg = fmt.Sprintf("[%s]%s ~ [white]%s", c.cfg.UserColour, c.cfg.Username, msg)
	c.chatView.Write([]byte(msg))
}
