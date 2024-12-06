package client

import (
	"crypto/rsa"
	"log"
	"net"

	"github.com/MatthewTully/simple-chat-server/internal/crypto"
	"github.com/MatthewTully/simple-chat-server/internal/encoding"
	"github.com/MatthewTully/simple-chat-server/internal/server"
	"github.com/rivo/tview"
)

type ClientConfig struct {
	Username     string `json:"username"`
	UserColour   string `json:"user_colour"`
	Logger       *log.Logger
	RSAKeyPair   crypto.RSAKeys
	ClientAESKey []byte
}

type Client struct {
	cfg             *ClientConfig
	ActiveConn      net.Conn
	Host            bool
	HostServer      *server.Server
	ServerAESKey    []byte
	ServerPubKey    *rsa.PublicKey
	processChannel  chan []byte
	LastCommand     string
	TUI             *tview.Application
	chatView        *tview.TextView
	activeUsersView *tview.TextView
	multiMessages   map[int]encoding.MsgProtocol
	userCmdArg      string
	tuiPages        *tview.Pages
	userInputBox    *tview.InputField
}

func NewClient(cfg *ClientConfig) Client {
	priv, pub, err := crypto.GenerateRSAKeyPair()
	if err != nil {
		cfg.Logger.Fatalf("could not generate key RSA pair for client: %v", err)
	}
	cfg.RSAKeyPair = crypto.RSAKeys{
		PrivateKey: priv,
		PublicKey:  pub,
	}

	aesKey, err := crypto.GenerateAESSecretKey()
	if err != nil {
		cfg.Logger.Fatalf("could not generate AES key for client: %v", err)
	}
	cfg.ClientAESKey = aesKey

	return Client{
		cfg:           cfg,
		multiMessages: make(map[int]encoding.MsgProtocol),
	}
}
