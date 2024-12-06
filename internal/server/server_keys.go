package server

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

func (s *Server) AwaitHandshake(conn net.Conn) (encoding.MsgProtocol, error) {
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				s.cfg.Logger.Printf("error reading from conn: %v\n", err)
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
			s.cfg.Logger.Print("handshake received")
			return dataPacket, nil
		}
	}
}

func (s *Server) SendHandshakeResponse(conn net.Conn) error {
	pubKeyBytes, err := crypto.RSAPublicKeyToBytes(s.cfg.RSAKeyPair.PublicKey)
	if err != nil {
		s.cfg.Logger.Printf("%v", err)
		return err
	}
	handshake, err := encoding.PrepHandshakeForSending(pubKeyBytes, s.cfg.ServerName, "white")
	if err != nil {
		return fmt.Errorf("error creating packet to send: %v", err)
	}
	s.cfg.Logger.Printf("SendHandshakeResponse: len %v\n", len(handshake))
	_, err = conn.Write(handshake)
	if err != nil {
		s.cfg.Logger.Printf("failed to send to user %s: %v\n", conn.RemoteAddr().String(), err)
		return fmt.Errorf("failed to send to user %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (s *Server) SendAESKey(conn net.Conn, cliPubKey *rsa.PublicKey) error {
	packet, err := encoding.PrepAESForSending(s.cfg.AESKey, cliPubKey, s.cfg.RSAKeyPair)
	if err != nil {
		return fmt.Errorf("failed to prepare AES Packet to send to user %s: %v", conn.RemoteAddr().String(), err)
	}
	s.cfg.Logger.Printf("SendAESKey: len %v\n", len(packet))
	s.cfg.Logger.Printf("SendAESKey: packet %v\n", packet)
	_, err = conn.Write(packet)
	if err != nil {
		s.cfg.Logger.Printf("failed to send to user %s: %v\n", conn.RemoteAddr().String(), err)
		return fmt.Errorf("failed to send to user %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (s *Server) AwaitClientAESKey(conn net.Conn, cliPubKey *rsa.PublicKey) ([]byte, error) {
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				s.cfg.Logger.Printf("error reading from conn: %v\n", err)
			}
			return nil, err
		}
		if nr == 0 {
			continue
		}
		data := buf[0:nr]

		sd := bytes.Split(data, encoding.HeaderPattern[:])
		encPacket := sd[1]
		packetLen := binary.BigEndian.Uint16(encPacket[4:])

		payload := encPacket[encoding.HeaderSize : packetLen+encoding.HeaderSize]
		buffer := bytes.NewBuffer(payload)
		dataPacket := encoding.DecodeAESPacket(buffer)

		decPayload, err := crypto.RSADecrypt(dataPacket.Data[:dataPacket.MsgSize], s.cfg.RSAKeyPair.PrivateKey)
		if err != nil {
			s.cfg.Logger.Printf("error Decrypting payload: %v", err)
			return nil, err
		}
		err = crypto.RSAVerify(decPayload, dataPacket.Sig[:dataPacket.SigSize], cliPubKey)
		if err != nil {
			s.cfg.Logger.Printf("error verifying payload: %v", err)
			return nil, err
		}

		if dataPacket.MessageType == encoding.SendAESKey {
			s.cfg.Logger.Print("AES Key Received complete")
			return decPayload, nil
		}
	}
}
