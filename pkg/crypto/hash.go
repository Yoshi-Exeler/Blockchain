package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func Hash(input string) ([]byte, error) {
	hash := sha256.New()
	_, err := hash.Write([]byte(input))
	if err != nil {
		return nil, fmt.Errorf("could not compute hash with error %v", err)
	}
	return hash.Sum(nil), nil
}

func HashB64(input string) (string, error) {
	hash := sha256.New()
	_, err := hash.Write([]byte(input))
	if err != nil {
		return "", fmt.Errorf("could not compute hash with error %v", err)
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}
