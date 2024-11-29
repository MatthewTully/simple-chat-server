package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func RSAPublicKeyToBytes(pubKey *rsa.PublicKey) ([]byte, error) {
	marshPub, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	pubBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: marshPub})
	return pubBytes, nil
}

func BytesToRSAPublicKey(pubBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pubBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid key bytes")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid key bytes: %v", err)
	}
	switch key := key.(type) {
	case *rsa.PublicKey:
		return key, nil
	default:
		return nil, fmt.Errorf("invalid key type")
	}
}
