package server

import (
	"fmt"
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

func (s *Server) AwaitMessage(conn net.Conn) {
	defer s.CloseConnection(conn)
	buf := make([]byte, s.Protocol.MaxSize)
	for {
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Printf("error reading from conn: %v\n", err)
			}
			return
		}
		if nr == 0 {
			return
		}

		data := buf[0:nr]
		//fmt.Printf("Inbound message from %v: %v", conn.RemoteAddr().String(), string(data))

		msg := []byte(fmt.Sprintf("[green]%v ~[white] ", conn.RemoteAddr().String()))
		msg = append(msg, data...)

		s.BroadcastMessage(conn.RemoteAddr().String(), msg)
	}
}

func SendMessage(conn net.Conn, msg []byte) error {
	_, err := conn.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to sent to user %s: %v", conn.RemoteAddr().String(), err)
	}
	return nil
}

func (s *Server) BroadcastMessage(sentBy string, message []byte) []error {
	failedAttempts := []error{}
	s.AddMsgToHistory(message)

	s.rwmu.RLock()
	defer s.rwmu.RUnlock()

	for users, conns := range s.LiveConns {
		if users != sentBy {
			err := SendMessage(conns, message)
			if err != nil {
				failedAttempts = append(failedAttempts, err)
			}
		}

	}
	if len(failedAttempts) > 0 {
		return failedAttempts
	}
	return nil
}

func (s *Server) SentMessageToClient(client string, msg []byte) error {
	s.rwmu.RLock()
	defer s.rwmu.RUnlock()
	user, ok := s.LiveConns[client]
	if !ok {
		return fmt.Errorf("failed to sent to user %s: User does not exist", user)
	}
	err := SendMessage(user, msg)
	return err
}

func (s *Server) ActionMessageType(p encoding.Protocol) error {
	switch p.MessageType {
	case encoding.KeepAlive:
		//update keep alive so user is not disconnected
	case encoding.Message:
		//Send send message to connected users

	case encoding.RequestConnect:
		// set a active connection
	case encoding.RequestDisconnect:
		// close connection
	}
	return fmt.Errorf("could not determine message type. %v", p.MessageType)
}
