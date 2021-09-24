package model

import (
	"coins/pkg/crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
)

const BlockReward = float64(1)
const BlockDiff = byte(1)
const EmptyBlockDiff = byte(1)
const THREADS = 4

type Block struct {
	ID            uint64         // Autoincrement id of the block
	Nonce         uint64         // Nonce to establish the required difficulty
	Hash          string         // Hash of this block
	Previous      string         // The Hash of the Previous block
	Miner         string         // The wallet address of the miner that received the block reward
	Transactions  []Transaction  // The Signed Transactions included in this block
	Registrations []Registration // The Registrations that happened in this block
}

func (b *Block) Mine(stop *bool) {
	var difficulty byte
	if len(b.Registrations) > 0 || len(b.Transactions) > 0 {
		difficulty = BlockDiff
	} else {
		difficulty = EmptyBlockDiff
	}
	signalChannel := make(chan uint64)
	for i := 0; i < THREADS; i++ {
		seed := uint64(rand.Uint32())<<32 + uint64(rand.Uint32())
		go mine(difficulty, seed, *b, signalChannel, stop)
	}
	result := <-signalChannel
	b.Nonce = result
	b.Hash, _ = b.GetHash()
	*stop = true
}

func mine(difficulty byte, seed uint64, block Block, sigChan chan uint64, stop *bool) {
	block.Nonce = seed
	for {
		if *stop || crypto.GetHashDiff(block.hashFast()) == difficulty {
			break
		}
		block.Nonce++
	}
	if !*stop {
		sigChan <- block.Nonce
	}
}

func (b *Block) hashFast() []byte {
	hash := []byte(fmt.Sprintf("%v%v%v%v", b.ID, b.Previous, b.Miner, b.Nonce))
	txBin, _ := json.Marshal(b.Transactions)
	regBin, _ := json.Marshal(b.Registrations)
	hash = append(hash, txBin...)
	hash = append(hash, regBin...)
	hasher := sha256.New()
	hasher.Write(hash)
	return hasher.Sum(nil)
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
	Wallet    string // The Wallet address of the user registering
	PublicKey string // the Public key of the user registering
}

type Transaction struct {
	TXID      uint64  // Autoincrement id for transactions
	Sender    string  // Wallet address of the sender
	Recipient string  // Wallet address of the recipient
	Amount    float64 // Amount of coins sent
	Comment   string  // Comment included with the transaction
	Hash      string  // The Hash of the Transaction
	Signature string  // The Signature of the Transaction hash made by the sender
}

func (tx *Transaction) Verify(publicKey *rsa.PublicKey) bool {
	decodedSignature, err := base64.StdEncoding.DecodeString(tx.Signature)
	if err != nil {
		return false
	}
	hash, err := tx.GetHash()
	if err != nil {
		return false
	}
	decodedHash := crypto.ToBytes(hash)
	if err != nil {
		return false
	}
	if !crypto.VerifySignature(decodedSignature, decodedHash, publicKey) {
		return false
	}
	return true
}

func (tx *Transaction) GetHash() (string, error) {
	hash := fmt.Sprintf("%v%v%v%v%v", tx.TXID, tx.Sender, tx.Recipient, tx.Amount, tx.Comment)
	return crypto.HashB64(hash)
}
