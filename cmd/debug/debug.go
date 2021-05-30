package main

import (
	"coins/pkg/blockchain"
	"coins/pkg/crypto"
	"coins/pkg/model"
	"encoding/base64"
	"fmt"
)

func main() {
	wal, err := blockchain.GenerateWalletFile()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Wallet:", wal)

	keyString, _ := blockchain.KeyToString(&wal.KP.PublicKey)

	bc := blockchain.BlockChain{Chainstate: blockchain.Chainstate{Wallets: make(map[string]*blockchain.WalletInfo)}, Blocks: []*model.Block{}}

	fmt.Printf("\nEmpty Blockchain:%+v\n", bc)

	firstBlock := model.Block{}
	firstBlock.Hash, _ = firstBlock.GetHash()

	bc.Blocks = append(bc.Blocks, &firstBlock)

	fmt.Printf("\nInsert first Block into Blockchain:%+v\n", bc)

	testTransaction := model.Transaction{
		GUID:      "GUID",
		Sender:    wal.Address,
		Recipient: "testwallet123",
		Amount:    0.1,
		Comment:   "Test Transaction",
	}

	testTransaction.Hash, _ = testTransaction.GetHash()

	decodedHash, _ := base64.StdEncoding.DecodeString(testTransaction.Hash)

	testTransaction.Signature, err = crypto.SignHashB64(decodedHash, wal.KP)
	if err != nil {
		fmt.Printf("Could not sign transaction with error %v\n", err)
	}

	fmt.Printf("\nTest Transaction:%+v\n", testTransaction)

	secondBlock := model.Block{
		ID:            1,
		Nonce:         0,
		Previous:      firstBlock.Hash,
		Miner:         wal.Address,
		Transactions:  []model.Transaction{testTransaction},
		Registrations: []model.Registration{{Wallet: "testwallet123", PublicKey: "testkey"}, {Wallet: wal.Address, PublicKey: keyString}},
	}

	secondBlock.Hash, _ = secondBlock.GetHash()

	fmt.Println("Hash Difficulty: ", crypto.GetHashDiff([]byte(secondBlock.Hash)))

	fmt.Println("Test Mining until diff = 4")

	secondBlock.Mine(3)

	fmt.Printf("\nSecond Block:%+v\n", secondBlock)

	bc.ProcessBlock(&secondBlock)

	bc.Print()

	bc.PrintWallets()

	bc.WriteToFile()
}
