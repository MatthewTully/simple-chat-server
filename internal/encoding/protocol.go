package encoding

import (
	"crypto/rsa"
	"encoding/binary"
	"log"
	"time"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
)

type MessageType uint8

const (
	MaxMessageSize                   = 1000
	MaxPacketSize                    = 1400
	HeaderSize                       = 6
	AESEncryptHeaderSize             = 2
	RequestConnect       MessageType = iota
	RequestDisconnect
	Message
	KeepAlive
	WhisperMessage
	ServerActiveUsers
	ErrorMessage
	SendAESKey
)

var HeaderPattern = [...]byte{0, 0, 27, 0, 5, 19, 93, 255, 255, 255}

type AESProtocol struct {
	MessageType MessageType
	MsgSize     uint16
	SigSize     uint16
	DateTime    time.Time
	Data        [crypto.EncodedKeySize]byte
	Sig         [crypto.EncodedKeySize]byte
}

type MsgProtocol struct {
	MessageType    MessageType
	MsgSize        uint16
	UsernameSize   uint16
	UserColourSize uint16
	Username       [32]byte
	UserColour     [32]byte
	DateTime       time.Time
	Data           [MaxMessageSize]byte
}

func setMsgProtocolUserFields(username, userColour string, p *MsgProtocol) {
	var userNameArr, userColourArr [32]byte

	usernameSlice := []byte(username)
	userColourSlice := []byte(userColour)
	copy(userNameArr[:], usernameSlice)
	copy(userColourArr[:], userColourSlice)

	p.Username = userNameArr
	p.UsernameSize = uint16(len(usernameSlice))
	p.UserColour = userColourArr
	p.UserColourSize = uint16(len(userColourSlice))
}

func PrepHandshakeForSending(msg []byte, sentFrom, colour string) ([]byte, error) {
	preppedBytes := []byte{}

	toSend := packageMessageBytes(msg)
	numPackets := uint16(len(toSend))
	for i, p := range toSend {
		p.MessageType = RequestConnect
		setMsgProtocolUserFields(sentFrom, colour, &p)
		dataPacket, err := encodePacket(p)
		if err != nil {
			return nil, err
		}
		packetLen := uint16(len(dataPacket.Bytes()))
		packetNum := uint16(i + 1)
		for _, i := range HeaderPattern {
			preppedBytes = append(preppedBytes, i)
		}

		preppedBytes = binary.BigEndian.AppendUint16(preppedBytes, packetNum)
		preppedBytes = binary.BigEndian.AppendUint16(preppedBytes, numPackets)
		preppedBytes = binary.BigEndian.AppendUint16(preppedBytes, packetLen)
		preppedBytes = append(preppedBytes, dataPacket.Bytes()...)

	}

	return preppedBytes, nil
}

func PrepAESForSending(key []byte, receiversPubKey *rsa.PublicKey, keyPair crypto.RSAKeys) ([]byte, error) {
	preppedBytes := []byte{}

	payloadSig, err := crypto.RSASign(key, keyPair.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	encryptedPayload, err := crypto.RSAEncrypt(key, receiversPubKey)
	if err != nil {
		log.Fatal(err)
	}

	toSend := packageAESBytes(encryptedPayload, payloadSig)
	numPackets := uint16(1)

	toSend.MessageType = SendAESKey
	dataPacket, err := encodePacket(toSend)
	if err != nil {
		return nil, err
	}
	packetLen := uint16(len(dataPacket.Bytes()))
	packetNum := uint16(1)
	for _, i := range HeaderPattern {
		preppedBytes = append(preppedBytes, i)
	}

	preppedBytes = binary.BigEndian.AppendUint16(preppedBytes, packetNum)
	preppedBytes = binary.BigEndian.AppendUint16(preppedBytes, numPackets)
	preppedBytes = binary.BigEndian.AppendUint16(preppedBytes, packetLen)
	preppedBytes = append(preppedBytes, dataPacket.Bytes()...)

	return preppedBytes, nil
}

func PrepBytesForSending(msg []byte, messageType MessageType, sentFrom, colour string, AESKey []byte) ([]byte, error) {
	preppedBytes := []byte{}

	toSend := packageMessageBytes(msg)
	numPackets := uint16(len(toSend))

	for i, p := range toSend {
		p.MessageType = messageType
		setMsgProtocolUserFields(sentFrom, colour, &p)
		dataPacket, err := encodePacket(p)
		if err != nil {
			return nil, err
		}
		packetLen := uint16(len(dataPacket.Bytes()))
		packetNum := uint16(i + 1)
		for _, i := range HeaderPattern {
			preppedBytes = append(preppedBytes, i)
		}

		payload := []byte{}
		payload = binary.BigEndian.AppendUint16(payload, packetNum)
		payload = binary.BigEndian.AppendUint16(payload, numPackets)
		payload = binary.BigEndian.AppendUint16(payload, packetLen)
		payload = append(payload, dataPacket.Bytes()...)

		encryptedPayload, err := crypto.AESEncrypt(payload, AESKey)
		if err != nil {
			log.Fatal(err)
		}
		encLen := uint16(len(encryptedPayload))

		preppedBytes = binary.BigEndian.AppendUint16(preppedBytes, encLen)
		preppedBytes = append(preppedBytes, encryptedPayload...)

	}

	return preppedBytes, nil
}
