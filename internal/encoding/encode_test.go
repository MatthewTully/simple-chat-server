package encoding

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	longTestString = `This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
1. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
2. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
3. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
4. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
5. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!`
)

func TestPackageMessageBytes(t *testing.T) {
	cases := []struct {
		name            string
		input           []byte
		expectedTotal   int
		expectedString  []string
		expectedMsgSize []uint16
	}{
		{
			name:          "input within size limit",
			input:         []byte("This byte string is within size"),
			expectedTotal: 1,
			expectedString: []string{
				"This byte string is within size",
			},
			expectedMsgSize: []uint16{31},
		}, {
			name:          "input over size limit",
			input:         []byte(longTestString),
			expectedTotal: 2,
			expectedString: []string{
				`This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
1. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
2. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
3. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
4. This byte string that is not wit`, `hin the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!
5. This byte string that is not within the size of MaxMessageSize. In fact it is over the limit, quite a bit over in fact. Just enough for two protocol packets to be sent, in fact, that is the expect result of this test. Of course that is a lot of bytes, so I'll just repeat this five times!`,
			},
			expectedMsgSize: []uint16{uint16(MaxMessageSize), 548},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := packageMessageBytes(tc.input)
			if len(got) != tc.expectedTotal {
				t.Errorf("Expected %v structs in slice. Got %v\n", tc.expectedTotal, len(got))
			}
			for i, protocols := range got {
				if len(protocols.Data) != MaxMessageSize {
					t.Errorf("Expected data size to be %v, Got %v\n", MaxMessageSize, len(protocols.Data))
				}
				if tc.expectedMsgSize[i] != protocols.MsgSize {
					t.Errorf("Expected protocol data size to be to be %v, Got %v\n", MaxMessageSize, len(protocols.Data))
				}
				if string(protocols.Data[:protocols.MsgSize]) != tc.expectedString[i] {
					t.Errorf("Expected protocol data [%v] %v to Equal %v\n", i, string(protocols.Data[:protocols.MsgSize]), tc.expectedString[i])
				}
			}
		})
	}
}

func TestPackageMessageString(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		expectedTotal int
	}{
		{
			name:          "short string message. 1 packet.",
			input:         "This is a short string.",
			expectedTotal: 1,
		}, {
			name:          "long string message. 2 packets.",
			input:         longTestString,
			expectedTotal: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := packageMessageString(tc.input)
			if tc.expectedTotal != len(got) {
				t.Errorf("Expected %v packets, Got %v\n", tc.expectedTotal, len(got))
			}
			var sb strings.Builder
			for _, protocols := range got {
				sb.Write(protocols.Data[:protocols.MsgSize])
			}
			reconstructedMsg := sb.String()
			if tc.input != reconstructedMsg {
				t.Errorf("Reconstructed String: Expected %v, Got %v\n", tc.input, reconstructedMsg)
			}

		})
	}
}

func TestEncodePacket(t *testing.T) {
	var userNameArr, userColourArr [32]byte

	usernameSlice := []byte("TestUser")
	userColourSlice := []byte("green")
	copy(userNameArr[:], usernameSlice)
	copy(userColourArr[:], userColourSlice)

	dataPacket := Protocol{
		MessageType:    KeepAlive,
		DateTime:       time.Now().UTC(),
		Data:           [MaxMessageSize]byte{},
		MsgSize:        0,
		Username:       userNameArr,
		UsernameSize:   uint16(len(usernameSlice)),
		UserColour:     userColourArr,
		UserColourSize: uint16(len(userColourSlice)),
	}
	encodedPacket := encodePacket(dataPacket)

	tmp := encodedPacket.Bytes()
	fmt.Printf("Len of packet: %v\n", len(tmp))

	decodedPacket := DecodePacket(bytes.NewBuffer(tmp))
	fmt.Printf("Decoded Packet.MessageType: %v\n", decodedPacket.MessageType)
	fmt.Printf("Decoded Packet.DateTime: %v\n", decodedPacket.DateTime)

	if decodedPacket != dataPacket {
		t.Errorf("decoded packet does not match original packet. Expected %v, Got %v", dataPacket, decodedPacket)
	}

}
