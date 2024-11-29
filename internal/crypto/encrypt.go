package crypto

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"
)

func RSAEncrypt(data []byte, key *rsa.PublicKey) ([]byte, error) {
	rand := rand.Reader
	hash := sha256.New()
	cipher, err := rsa.EncryptOAEP(hash, rand, key, data, nil)
	if err != nil {
		return nil, err
	}
	return cipher, nil
}

func RSADecrypt(cipher []byte, key *rsa.PrivateKey) ([]byte, error) {
	rand := rand.Reader
	hash := sha256.New()
	data, err := rsa.DecryptOAEP(hash, rand, key, cipher, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func RSASign(payload []byte, key *rsa.PrivateKey) ([]byte, error) {
	rand := rand.Reader
	hashed := sha256.Sum256(payload)
	sig, err := rsa.SignPKCS1v15(rand, key, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func RSAVerify(payload, sigBytes []byte, key *rsa.PublicKey) error {
	hashed := sha256.Sum256(payload)
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], sigBytes)
}

func AESEncrypt(payload, aesKey []byte) ([]byte, error) {
	ciBlock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(ciBlock)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(iv, iv, payload, nil), nil
}

func AESDecrypt(payload, aesKey []byte) ([]byte, error) {
	ciBlock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(ciBlock)
	if err != nil {
		return nil, err
	}

	ivSize := gcm.NonceSize()
	if len(payload) < ivSize {
		return nil, fmt.Errorf("encrypted bytes smaller than expected NONCE size")
	}

	iv, cipherBytes := payload[:ivSize], payload[ivSize:]
	return gcm.Open(nil, iv, cipherBytes, nil)

}
