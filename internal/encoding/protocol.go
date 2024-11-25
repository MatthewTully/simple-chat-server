package encoding

import "time"

type MessageType int

const (
	MaxMessageSize             = 1200
	RequestConnect MessageType = iota
	RequestDisconnect
	Message
	KeepAlive
)

type Protocol struct {
	PacketNun    uint16
	TotalPackets uint16
	MsgSize      uint16
	MessageType  MessageType
	DateTime     time.Time
	Username     [32]byte
	UserColour   [32]byte
	Data         [MaxMessageSize]byte
}
