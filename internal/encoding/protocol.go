package encoding

import (
	"encoding/binary"
	"time"
)

type MessageType uint8

const (
	MaxMessageSize             = 1000
	MaxPacketSize              = 1400
	HeaderSize                 = 6
	RequestConnect MessageType = iota
	RequestDisconnect
	Message
	KeepAlive
	WhisperMessage
	ServerActiveUsers
)

var HeaderPattern = [...]byte{0, 0, 27, 0, 5, 19, 93, 255, 255, 255}

type Protocol struct {
	MessageType    MessageType
	MsgSize        uint16
	UsernameSize   uint16
	UserColourSize uint16
	Username       [32]byte
	UserColour     [32]byte
	DateTime       time.Time
	Data           [MaxMessageSize]byte
}

func setProtocolUserFields(username, userColour string, p *Protocol) {
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

func PrepBytesForSending(msg []byte, messageType MessageType, sentFrom, colour string) []byte {
	preppedBytes := []byte{}

	toSend := packageMessageBytes(msg)
	numPackets := uint16(len(toSend))

	for i, p := range toSend {
		p.MessageType = messageType
		setProtocolUserFields(sentFrom, colour, &p)
		dataPacket := encodePacket(p)
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
	//fmt.Printf("%v\n", preppedBytes)

	return preppedBytes
}
