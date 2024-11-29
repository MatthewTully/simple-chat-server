package encoding

import (
	"bytes"
	"encoding/gob"
	"log"
)

func DecodeMsgPacket(buffer *bytes.Buffer) MsgProtocol {

	var packet MsgProtocol
	dec := gob.NewDecoder(buffer)

	err := dec.Decode(&packet)
	if err != nil {
		log.Print(err)
	}
	return packet

}

func DecodeAESPacket(buffer *bytes.Buffer) AESProtocol {

	var packet AESProtocol
	dec := gob.NewDecoder(buffer)

	err := dec.Decode(&packet)
	if err != nil {
		log.Print(err)
	}
	return packet

}
