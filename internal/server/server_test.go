package server

import (
	"fmt"
	"net"
	"testing"
)

func TestAddMessageToHistory(t *testing.T) {
	cases := []struct {
		name          string
		inputCount    int
		expectedTotal int
		setLimit      int
		expectedMsgs  [][]byte
	}{
		{
			name:          "less than the max",
			inputCount:    10,
			expectedTotal: 10,
			setLimit:      20,
			expectedMsgs:  [][]byte{},
		}, {
			name:          "exact limit",
			inputCount:    15,
			expectedTotal: 15,
			setLimit:      15,
			expectedMsgs:  [][]byte{},
		}, {
			name:          "move than the limit",
			inputCount:    50,
			expectedTotal: 20,
			setLimit:      20,
			expectedMsgs:  [][]byte{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, err := NewServer("8144", uint(tc.setLimit))
			srv.Listener.Close()
			if err != nil {
				t.Errorf("error declaring srv for test case %v, error: %v", tc.name, err)
			}
			for i := range tc.inputCount {
				msg := []byte(fmt.Sprintf("%d", i))
				if i >= (tc.inputCount - tc.setLimit) {
					tc.expectedMsgs = append(tc.expectedMsgs, msg)
				}
				srv.AddMsgToHistory(msg)
			}

			if len(srv.MsgHistory) != tc.expectedTotal {
				t.Errorf("Expected MsgHistory to contain %d elements. Contained %v", tc.expectedTotal, len(srv.MsgHistory))
			}
			for i, msg := range srv.MsgHistory {
				if string(msg) != string(tc.expectedMsgs[i]) {
					t.Errorf("Message history does not match. Expected %v at index %d, Got %v", string(tc.expectedMsgs[i]), i, string(msg))
				}
			}
		})
	}
}

func TestConnectionLimits(t *testing.T) {
	cases := []struct {
		name                string
		connCount           uint
		connectionLimit     uint
		expectedConnections uint
		expectedKeys        []string
	}{
		{
			name:                "conns within limit",
			connCount:           5,
			connectionLimit:     20,
			expectedConnections: 5,
			expectedKeys:        []string{},
		},
		{
			name:                "conns at limit",
			connCount:           10,
			connectionLimit:     10,
			expectedConnections: 10,
			expectedKeys:        []string{},
		}, {
			name:                "conns over limit",
			connCount:           25,
			connectionLimit:     15,
			expectedConnections: 15,
			expectedKeys:        []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, err := NewServer("8144", 10)
			srv.MaxConnectionLimit = tc.connectionLimit
			if err != nil {
				t.Errorf("error declaring srv for test case %v, error: %v", tc.name, err)
			}
			for i := range tc.connCount {
				conn, err := net.Dial("tcp", ":8144")
				if err != nil {
					t.Errorf("error creating conn in test %v", tc.name)
				}
				if i < tc.expectedConnections {
					tc.expectedKeys = append(tc.expectedKeys, fmt.Sprintf("%v", i))
				}
				err = srv.AddToLiveConns(fmt.Sprintf("%v", i), conn)
				if err != nil {
					fmt.Println(err)
				}
			}
			srv.Listener.Close()

			if tc.expectedConnections != uint(len(srv.LiveConns)) {
				t.Errorf("Expected LiveConns to contain %d elements. Contained %v", tc.expectedConnections, len(srv.LiveConns))
			}
			for _, key := range tc.expectedKeys {
				conn, ok := srv.LiveConns[key]
				if !ok {
					t.Errorf("Expected Key %v to be in the LiveConns, but it was not.", key)
				}
				conn.Close()

			}
		})
	}
}
