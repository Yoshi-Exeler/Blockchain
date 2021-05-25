package blockchain

import (
	"coins/pkg/crypto"
	"coins/pkg/model"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

type BlockChain struct {
	Blocks            []*model.Block
	Wallets           map[string]*WalletInfo // map[WalletAddress]Currency
	LastBlock         *model.Block
	MarketVolume      float64
	TransactionVolume uint64
}

type WalletInfo struct {
	Amount    float64
	PublicKey string
}

func (bc *BlockChain) Print() {
	fmt.Println("-----------------------")
	fmt.Printf("|      BlockChain     |\n")
	fmt.Printf("|Market:%v             |\n", bc.MarketVolume)
	fmt.Printf("|Tx:%v                 |\n", bc.TransactionVolume)
	fmt.Printf("|Blocks:%v             |\n", len(bc.Blocks))
	fmt.Printf("|LastBlock:%v          |\n", bc.LastBlock.ID)
	fmt.Println("-----------------------")
}

func (bc *BlockChain) PrintWallets() {
	for key, value := range bc.Wallets {
		fmt.Printf("%v :: %v Coins\n", key, value.Amount)
	}
}

func (bc *BlockChain) ProcessAll() {
	// Reset the Blockchain
	bc.Wallets = make(map[string]*WalletInfo)
	bc.LastBlock = nil
	bc.MarketVolume = 0
	bc.TransactionVolume = 0
	// If transaction blocks actually exist
	if len(bc.Blocks) > 1 {
		// Process all the blocks
		for _, block := range bc.Blocks[1:] {
			alloc := *block
			// Validate the Current Block
			if !bc.ValidateBlock(&alloc) {
				log.Printf("[BlockChain] Block %v is invalid and will be skipped\n", alloc.ID)
				continue
			}
			// Process the current block
			bc.ProcessBlock(&alloc)
		}
	}
}

func (bc *BlockChain) ValidateBlock(b *model.Block) bool {
	// Check that this block is a valid next block
	if b.Previous != bc.LastBlock.Hash {
		return false
	}
	// Verify that the block is generally a valid Block
	if !VerifyBlock(b) {
		// if the block is invalid, we just skip it
		return false
	}
	// Check all transaction signatures
	for _, tx := range b.Transactions {
		decodedSignature, _ := base64.StdEncoding.DecodeString(tx.Signature)
		hash, _ := b.GetHash()
		decodedHash, _ := base64.StdEncoding.DecodeString(hash)
		key, _ := crypto.StringToKey(bc.Wallets[tx.Sender].PublicKey)
		if !crypto.VerifySignature(decodedSignature, decodedHash, key) {
			return false
		}
	}
	return true
}

func (bc *BlockChain) ProcessBlock(b *model.Block) error {
	// Process the Registrations in this block
	for _, reg := range b.Registrations {
		bc.Wallets[reg.Wallet] = &WalletInfo{}
		bc.Wallets[reg.Wallet].Amount = 0
		bc.Wallets[reg.Wallet].PublicKey = reg.PublicKey
	}
	// Add the Miners fee to the miners wallet
	bc.Wallets[b.Miner].Amount += model.BlockReward
	bc.MarketVolume += model.BlockReward
	// Process the Transactions
	for _, tx := range b.Transactions {
		bc.Wallets[tx.Sender].Amount -= tx.Amount
		bc.Wallets[tx.Recipient].Amount += tx.Amount
		bc.TransactionVolume++
	}
	// Set the Lastblock to the processed block
	bc.LastBlock = b
	// Append the Block
	bc.Blocks = append(bc.Blocks, b)
	return nil
}

func VerifyBlock(block *model.Block) bool {
	// We dont allow empty blocks
	if len(block.Registrations) == 0 && len(block.Transactions) == 0 {
		return false
	}
	// We dont accept blocks that miss vital parameters
	if len(block.Hash) == 0 || len(block.Miner) == 0 || len(block.Previous) == 0 {
		return false
	}
	// We dont accept blocks with id 0
	if block.ID == 0 {
		return false
	}
	return true
}

func ReadFile() (*BlockChain, error) {
	// Read the blockchain file
	bin, err := ioutil.ReadFile("sjc.blockchain")
	if err != nil {
		return nil, fmt.Errorf("could not read blockchain with error %v", err)
	}
	// unmarshall the blockchain
	var bcf BlockChain
	err = json.Unmarshal(bin, &bcf)
	if err != nil {
		return nil, fmt.Errorf("could not deserialize blockchain with error %v", err)
	}
	return &bcf, err
}

func (bc *BlockChain) WriteToFile() error {
	// marshall the blockchain
	bin, err := json.Marshal(bc)
	if err != nil {
		return fmt.Errorf("could not serialize blockchain with error %v", err)
	}
	// write the blockchain to the file
	err = ioutil.WriteFile("sjc.blockchain", bin, 0644)
	if err != nil {
		return fmt.Errorf("could not wrtie blockchain with error %v", err)
	}
	return nil
}