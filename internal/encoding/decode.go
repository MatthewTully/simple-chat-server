package encoding

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

func DecodePacket(buffer *bytes.Buffer) Protocol {

	var packet Protocol
	dec := gob.NewDecoder(buffer)

	err := dec.Decode(&packet)
	if err != nil {
		fmt.Println(err)
	}
	return packet

}
