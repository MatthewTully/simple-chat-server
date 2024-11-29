package crypto

import (
	"crypto/rand"
	"crypto/rsa"
)

const (
	bitSize        = 2048
	AESKeySize     = 32
	EncodedKeySize = 256
)

type RSAKeys struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func GenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	rand := rand.Reader
	priv, err := rsa.GenerateKey(rand, bitSize)
	if err != nil {
		return nil, nil, err
	}

	err = priv.Validate()
	if err != nil {
		return nil, nil, err
	}

	return priv, &priv.PublicKey, nil
}

func GenerateAESSecretKey() ([]byte, error) {
	keyBytes := make([]byte, AESKeySize)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return nil, err
	}
	return keyBytes, nil
}
