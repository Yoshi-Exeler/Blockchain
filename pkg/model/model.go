package model

import (
	"coins/pkg/crypto"
	"encoding/json"
	"fmt"
)

const BlockReward = float64(1)

type Block struct {
	ID            uint64         // Autoincrement id of the block
	Nonce         uint64         // Nonce to establish the required difficulty
	Hash          string         // Hash of this block
	Previous      string         // The Hash of the Previous block
	Miner         string         // The wallet address of the miner that received the block reward
	Transactions  []Transaction  // The Signed Transactions included in this block
	Registrations []Registration // The Registrations that happened in this block
}

func (b *Block) GetHash() (string, error) {
	hash := fmt.Sprintf("%v%v%v%v", b.ID, b.Previous, b.Miner, b.Nonce)
	txBin, err := json.Marshal(b.Transactions)
	if err != nil {
		return "", err
	}
	regBin, err := json.Marshal(b.Registrations)
	if err != nil {
		return "", err
	}
	hash += string(txBin)
	hash += string(regBin)
	return crypto.HashB64(hash)
}

type Registration struct {
	Wallet     string // The Wallet address of the user registering
	PublicKey  string // the Public key of the user registering
	CommonName string // The (Optional) Common name of the user registering
}

type Transaction struct {
	GUID      string  // Randomly generated id for the transaction
	Sender    string  // Wallet address of the sender
	Recipient string  // Wallet address of the recipient
	Amount    float64 // Amount of coins sent
	Comment   string  // Comment included with the transaction
	Hash      string  // The Hash of the Transaction
	Signature string  // The Signature of the Transaction hash made by the sender
}

func (tx *Transaction) GetHash() (string, error) {
	hash := fmt.Sprintf("%v%v%v%v%v", tx.GUID, tx.Sender, tx.Recipient, tx.Amount, tx.Comment)
	return crypto.HashB64(hash)
}
