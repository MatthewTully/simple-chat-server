package encoding

import "time"

const (
	MaxMessageSize = 1200
)

type Protocol struct {
	PacketNun    uint16
	TotalPackets uint16
	MsgSize      uint16
	DateTime     time.Time
	Username     [32]byte
	UserColour   [32]byte
	Data         [MaxMessageSize]byte
}
