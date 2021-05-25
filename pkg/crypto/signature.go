package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
)

func SignHash(hash []byte, kp *rsa.PrivateKey) ([]byte, error) {
	signature, err := rsa.SignPSS(rand.Reader, kp, crypto.SHA256, hash, nil)
	if err != nil {
		return nil, fmt.Errorf("could not sign hash with error %v", err)
	}
	return signature, nil
}

func SignHashB64(hash []byte, kp *rsa.PrivateKey) (string, error) {
	signature, err := rsa.SignPSS(rand.Reader, kp, crypto.SHA256, hash, nil)
	if err != nil {
		return "", fmt.Errorf("could not sign hash with error %v", err)
	}
	return base64.URLEncoding.EncodeToString(signature), nil
}

func VerifySignature(signature []byte, hash []byte, publicKey *rsa.PublicKey) bool {
	return rsa.VerifyPSS(publicKey, crypto.SHA256, hash, signature, nil) == nil
}
