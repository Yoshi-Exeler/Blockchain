package blockchain

import (
	"coins/pkg/crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func KeyToString(pub *rsa.PublicKey) (string, error) {
	// json marshall the public key
	bin, err := json.Marshal(pub)
	if err != nil {
		return "", fmt.Errorf("could not serialize public key with error %v", err)
	}
	// Base64 encode the json bytes
	return base64.URLEncoding.EncodeToString([]byte(bin)), nil
}

func StringToKey(addr string) (*rsa.PublicKey, error) {
	// Base64 decode the address
	decoded, err := base64.StdEncoding.DecodeString(addr)
	if err != nil {
		return nil, fmt.Errorf("could not decode public key with error %v", err)
	}
	// Unmarshall the public key
	var pub rsa.PublicKey
	err = json.Unmarshal(decoded, &pub)
	if err != nil {
		return nil, fmt.Errorf("could not deserialize public key with error %v", err)
	}
	return &pub, nil
}

func GenerateWalletFile() (*Wallet, error) {
	// Declare a new wallet
	wallet := Wallet{}
	// Generate a 2048 Bit RSA Keypair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair with error %v", err)
	}
	// Set the Keypair
	wallet.KP = privateKey
	// Generate a wallet address
	wallet.Address = crypto.RandomString(32)
	// Serialize the Wallet to json
	bin, err := json.Marshal(wallet)
	if err != nil {
		return nil, fmt.Errorf("could not serialize wallet with error %v", err)
	}
	// Write the Wallet to a file
	err = ioutil.WriteFile("wallet.json", bin, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not write wallet to file with error %v", err)
	}
	return &wallet, nil
}

func ReadWalletFile() (*Wallet, error) {
	// Read the Keypair file
	bin, err := ioutil.ReadFile("wallet.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file with error %v", err)
	}
	// Unmarshall the Keypair
	var wallet Wallet
	err = json.Unmarshal(bin, &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize wallet with error %v", err)
	}
	return &wallet, nil
}

type Wallet struct {
	KP      *rsa.PrivateKey // The keypair for this wallet
	Address string          // The address of the wallet
}
