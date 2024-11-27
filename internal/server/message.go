package server

import (
	"fmt"
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/encoding"
)

func (s *Server) ActionMessageType(p encoding.Protocol) error {
	switch p.MessageType {
	case encoding.KeepAlive:
		//update keep alive so user is not disconnected
	case encoding.Message:
		sentBy := string(p.Username[:p.UsernameSize])
		msg := []byte(fmt.Sprintf("[%s]%v ~[white] ", string(p.UserColour[:p.UserColourSize]), sentBy))
		msg = append(msg, p.Data[:p.MsgSize]...)
		s.ProcessGroupMessage(sentBy, msg)
	case encoding.WhisperMessage:
		//Sent to user
	case encoding.RequestDisconnect:
		s.CloseConnectionForUser(string(p.Username[:p.UsernameSize]))
	}
	return fmt.Errorf("could not determine message type. %v", p.MessageType)
}

func (s *Server) ActionMessageTypeMultiMessage(p encoding.Protocol, data []byte) error {
	switch p.MessageType {
	case encoding.KeepAlive:
		//update keep alive so user is not disconnected
	case encoding.Message:
		sentBy := string(p.Username[:p.UsernameSize])
		msg := []byte(fmt.Sprintf("[%s]%v ~[white] ", string(p.UserColour[:p.UserColourSize]), sentBy))
		msg = append(msg, data...)
		s.ProcessGroupMessage(sentBy, msg)
	case encoding.WhisperMessage:
		//Sent to user
	case encoding.RequestDisconnect:
		s.CloseConnectionForUser(string(p.Username[:p.UsernameSize]))
	}
	return fmt.Errorf("could not determine message type. %v", p.MessageType)
}

func (s *Server) ProcessGroupMessage(sentBy string, msg []byte) {
	s.AddMsgToHistory(msg)
	toSend := encoding.PrepBytesForSending(msg, encoding.Message, s.cfg.ServerName, "white")
	s.cfg.Logger.Printf("ProcessGroupMessage: len %v\n", len(toSend))
	s.BroadcastMessage(sentBy, toSend)
}

func (s *Server) AwaitMessage(user ConnectedUser) {
	defer s.CloseConnection(user)
	for {
		buf := make([]byte, encoding.MaxPacketSize)
		var data []byte

		nr, err := user.conn.Read(buf)
		s.cfg.Logger.Printf("Server: nr=%v\n", nr)

		data = buf[0:nr]

		if nr == 0 {
			return
		}

		if err != nil {
			s.cfg.Logger.Printf("Server error\n")
			if err.Error() != "EOF" {
				s.cfg.Logger.Printf("error reading from conn: %v\n", err)
			}
			return
		}
		user.processChannel <- data

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

	s.rwmu.RLock()
	defer s.rwmu.RUnlock()

	for users, conns := range s.LiveConns {
		if users != sentBy {
			err := SendMessage(conns.conn, message)
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
		return fmt.Errorf("failed to sent to user %s: User does not exist", user.userInfo.Username)
	}

	toSend := encoding.PrepBytesForSending(msg, encoding.Message, s.cfg.ServerName, "white")

	s.cfg.Logger.Printf("SentMessageToClient: len %v\n", len(toSend))
	err := SendMessage(user.conn, toSend)
	return err
}
