package server

import (
	"log"
	"net"
	"sync"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
)

type serverConfig struct {
	ServerName string
	HostUser   string
	Logger     *log.Logger
	RSAKeyPair crypto.RSAKeys
	AESKey     []byte
}

type Server struct {
	cfg                *serverConfig
	LiveConns          map[string]*ConnectedUser
	Listener           net.Listener
	MsgHistory         [][]byte
	MaxMsgHistorySize  uint
	MaxConnectionLimit uint
	Blacklist          []string
	rwmu               *sync.RWMutex
}

func NewServer(port string, historySize uint, logger *log.Logger) (Server, error) {
	l, err := NewListener(port)
	if err != nil {
		return Server{}, err
	}

	priv, pub, err := crypto.GenerateRSAKeyPair()
	if err != nil {
		logger.Fatalf("could not generate RSA key pair for server: %v", err)
	}

	aesKey, err := crypto.GenerateAESSecretKey()
	if err != nil {
		logger.Fatalf("could not generate AES key for server: %v", err)
	}

	srvCfg := serverConfig{
		ServerName: "Chat Server",
		Logger:     logger,
		RSAKeyPair: crypto.RSAKeys{
			PrivateKey: priv,
			PublicKey:  pub,
		},
		AESKey: aesKey,
	}

	srv := Server{
		LiveConns:         make(map[string]*ConnectedUser),
		Listener:          l,
		cfg:               &srvCfg,
		MsgHistory:        [][]byte{},
		MaxMsgHistorySize: historySize,
		rwmu:              &sync.RWMutex{},
	}
	return srv, nil
}

func (s *Server) SetHostUser(username string) {
	s.cfg.HostUser = username
}
