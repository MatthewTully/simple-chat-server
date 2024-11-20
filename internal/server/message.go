package server

import (
	"fmt"
	"net"
)

func (s *Server) AwaitMessage(conn net.Conn) {
	defer s.CloseConnection(conn)
	buf := make([]byte, s.Protocol.MaxSize)
	for {
		nr, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("error reading from conn: %v\n", err)
			return
		}
		if nr == 0 {
			return
		}

		data := buf[0:nr]
		fmt.Printf("Inbound message from %v: %v", conn.RemoteAddr().String(), string(data))

		msg := []byte(fmt.Sprintf("\033[32m%v:\033[0m", conn.RemoteAddr().String()))
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
