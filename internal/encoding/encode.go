package encoding

import (
	"bytes"
	"encoding/gob"
	"time"
	"unsafe"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
)

func encodePacket(packet any) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(make([]byte, 0, unsafe.Sizeof(packet)))
	enc := gob.NewEncoder(buf)

	err := enc.Encode(packet)
	if err != nil {
		return nil, err
	}
	return buf, nil

}

func packageMessageString(msg string) []MsgProtocol {
	msgBytes := []byte(msg)
	return packageMessageBytes(msgBytes)
}

func packageMessageBytes(msg []byte) []MsgProtocol {
	lengthMessage := len(msg)
	protocolSlice := []MsgProtocol{}
	if lengthMessage > MaxMessageSize {
		tmpSlice := packageMessageBytes(msg[:MaxMessageSize])
		msg = msg[MaxMessageSize:lengthMessage]
		protocolSlice = append(protocolSlice, tmpSlice...)
		lengthMessage = len(msg)
	}
	var m [MaxMessageSize]byte
	copy(m[:], msg)
	newProtocol := MsgProtocol{
		DateTime: time.Now().UTC(),
		Data:     m,
		MsgSize:  uint16(lengthMessage),
	}
	protocolSlice = append(protocolSlice, newProtocol)
	return protocolSlice
}

func packageAESBytes(msg []byte, sig []byte) AESProtocol {
	lengthMessage := uint16(len(msg))
	lengthSig := uint16(len(sig))

	var m [crypto.EncodedKeySize]byte
	var s [crypto.EncodedKeySize]byte
	copy(m[:], msg)
	copy(s[:], sig)
	newProtocol := AESProtocol{
		DateTime: time.Now().UTC(),
		Data:     m,
		MsgSize:  lengthMessage,
		Sig:      s,
		SigSize:  lengthSig,
	}
	return newProtocol
}
