package server

import (
	"fmt"
	"log"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

const (
	network = "tcp"
)

func NewListener(port string) (net.Listener, error) {

	addr := fmt.Sprintf(":%s", port)

	listener, err := net.Listen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("error creating Listener: %v", err)
	}
	return listener, nil
}

func (s *Server) StartListening() {
	s.cfg.Logger.Printf("Server is listening on %v\n", s.Listener.Addr().String())
	defer s.Listener.Close()
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		conIp := strings.Split(conn.RemoteAddr().String(), ":")[0]
		if slices.Contains(s.Blacklist, conIp) {
			s.DenyConnection(conn, "cannot connect to server: IP banned")
		}

		c := make(chan ConnectedUser, 1)
		go func() {
			cliPub, err := s.AwaitHandshake(conn)
			if err != nil {
				s.DenyConnection(conn, err.Error())
				return
			}
			key, err := crypto.BytesToRSAPublicKey(cliPub.Data[:cliPub.MsgSize])
			if err != nil {
				s.DenyConnection(conn, err.Error())
				return
			}

			err = s.SendHandshakeResponse(conn)
			if err != nil {
				s.DenyConnection(conn, err.Error())
				return
			}

			cliAES, err := s.AwaitClientAESKey(conn, key)
			if err != nil {
				s.DenyConnection(conn, err.Error())
				return
			}

			err = s.SendAESKey(conn, key)
			if err != nil {
				s.DenyConnection(conn, err.Error())
				return
			}
			newUser := ConnectedUser{
				conn: conn,
				userInfo: struct {
					Username   string
					UserColour string
				}{
					Username:   string(cliPub.Username[:cliPub.UsernameSize]),
					UserColour: string(cliPub.UserColour[:cliPub.UserColourSize]),
				},
				publicKey:     key,
				AESKey:        cliAES,
				multiMessages: make(map[int]encoding.MsgProtocol),
			}
			if err != nil {
				s.DenyConnection(conn, err.Error())
			}
			c <- newUser
		}()

		select {
		case res := <-c:
			user, err := s.NewConnection(res)
			if err != nil {
				s.DenyConnection(conn, err.Error())
			} else {
				go user.ProcessMessage(s)
			}
		case <-time.After(30 * time.Second):
			s.DenyConnection(conn, "cannot connect to server: Connection timed out")
		}

	}
}
