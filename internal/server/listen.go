package server

import (
	"fmt"
	"log"
	"net"
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
	fmt.Printf("Server is listening on %v\n", s.Listener.Addr().String())
	defer s.Listener.Close()
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		err = s.NewConnection(conn)
		if err != nil {
			s.DenyConnection(conn, err.Error())
		} else {
			go s.AwaitMessage(conn)
		}
	}
}
