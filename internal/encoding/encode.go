package encoding

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
	"unsafe"
)

func encodePacket(packet Protocol) *bytes.Buffer {
	fmt.Println("starting encode.")
	buf := bytes.NewBuffer(make([]byte, 0, unsafe.Sizeof(packet)))
	enc := gob.NewEncoder(buf)

	err := enc.Encode(packet)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished encode.")
	return buf

}

func packageMessageString(msg string) []Protocol {
	msgBytes := []byte(msg)
	return packageMessageBytes(msgBytes)
}

func packageMessageBytes(msg []byte) []Protocol {
	lengthMessage := len(msg)
	protocolSlice := []Protocol{}
	if lengthMessage > MaxMessageSize {
		tmpSlice := packageMessageBytes(msg[:MaxMessageSize])
		msg = msg[MaxMessageSize:lengthMessage]
		protocolSlice = append(protocolSlice, tmpSlice...)
		lengthMessage = len(msg)
	}
	var ms [MaxMessageSize]byte
	copy(ms[:], msg)
	newProtocol := Protocol{
		DateTime: time.Now().UTC(),
		Data:     ms,
		MsgSize:  uint16(lengthMessage),
	}
	protocolSlice = append(protocolSlice, newProtocol)
	return protocolSlice
}
