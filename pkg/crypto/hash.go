package crypto

import (
	"crypto/sha256"
	"encoding/hex"
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
	//return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ToBytes(hash string) []byte {
	bytes, _ := hex.DecodeString(hash)
	return bytes
}

func GetHashDiff(hash []byte) uint8 {
	diff := uint8(0)
	for _, hashByte := range hash {
		if hashByte != 0 {
			break
		}
		diff++
	}
	return diff
}
