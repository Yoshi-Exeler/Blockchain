package blockchain

import (
	"coins/pkg/crypto"
	"coins/pkg/model"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

type BlockChain struct {
	Blocks     []*model.Block
	Chainstate Chainstate
}

type Chainstate struct {
	Wallets           map[string]*WalletInfo // map[WalletAddress]Currency
	LastBlock         model.Block
	MarketVolume      float64
	TransactionVolume uint64
}

type WalletInfo struct {
	TXC       uint64 // Transaction Counter to avoid transaction duplication attacks
	Amount    float64
	PublicKey string
}

func (bc *BlockChain) Print() {
	fmt.Println("-----------------------")
	fmt.Printf("|      BlockChain     |\n")
	fmt.Printf("|Market:%v             |\n", bc.Chainstate.MarketVolume)
	fmt.Printf("|Tx:%v                 |\n", bc.Chainstate.TransactionVolume)
	fmt.Printf("|Blocks:%v             |\n", len(bc.Blocks))
	fmt.Printf("|LastBlock:%v          |\n", bc.Chainstate.LastBlock.ID)
	fmt.Println("-----------------------")
}

func (bc *BlockChain) PrintWallets() {
	for key, value := range bc.Chainstate.Wallets {
		fmt.Printf("%v :: %v Coins\n", key, value.Amount)
	}
}

func (bc *BlockChain) ProcessAll() {
	// Reset the Blockchain
	bc.Chainstate.Wallets = make(map[string]*WalletInfo)
	bc.Chainstate.MarketVolume = 0
	bc.Chainstate.TransactionVolume = 0
	// If transaction blocks actually exist
	if len(bc.Blocks) > 1 {
		// Process all the blocks
		for _, block := range bc.Blocks[1:] {
			alloc := *block
			// Validate the Current Block
			res := bc.ValidateBlock(alloc)
			if res != B_ACCEPT {
				log.Printf("[BlockChain] Block %v is invalid and will be skipped, reason=%v\n", alloc.ID, res)
				continue
			}
			// Process the current block
			bc.ProcessBlock(alloc)
		}
	}
}

type BLOCK_VALIDATION_RESULT string

const (
	B_ACCEPT               = BLOCK_VALIDATION_RESULT("BLOCK_ACCEPT")
	B_REJECT_HASH_INTEG    = BLOCK_VALIDATION_RESULT("BLOCK_REJECT_NO_HASH_SEQUENCE_INTEGRITY")
	B_REJECT_ID_INTEG      = BLOCK_VALIDATION_RESULT("BLOCK_REJECT_NO_ID_SEQUENCE_INTEGRITY")
	B_REJECT_WRONG_DIFF    = BLOCK_VALIDATION_RESULT("BLOCK_REJECT_WRONG_HASH_DIFF")
	B_REJECT_BLOCK_INVALID = BLOCK_VALIDATION_RESULT("BLOCK_REJECT_BLOCK_INVALID")
	B_REJECT_TX_INVALID    = BLOCK_VALIDATION_RESULT("BLOCK_REJECT_TRANSACTION_INVALID")
)

func (bc *BlockChain) ValidateBlock(b model.Block) BLOCK_VALIDATION_RESULT {
	// Check that this block is a valid next block
	if b.Previous != bc.Chainstate.LastBlock.Hash {
		return B_REJECT_HASH_INTEG
	}
	// Check that the id was incremented correctly
	if bc.Chainstate.LastBlock.ID+1 != b.ID {
		return B_REJECT_ID_INTEG
	}
	// Check that the block has the correct difficulty
	if len(b.Transactions) > 0 || len(b.Registrations) > 0 {
		if crypto.GetHashDiff(crypto.ToBytes(b.Hash)) != model.BlockDiff {
			return B_REJECT_WRONG_DIFF
		}
	} else {
		if crypto.GetHashDiff(crypto.ToBytes(b.Hash)) != model.EmptyBlockDiff {
			return B_REJECT_WRONG_DIFF
		}
	}
	// Verify that the block is generally a valid Block
	if !VerifyBlock(b) {
		// if the block is invalid, we just skip it
		return B_REJECT_BLOCK_INVALID
	}
	// Check all transaction signatures
	for _, tx := range b.Transactions {
		// find the public key of the sender
		key, err := StringToKey(bc.Chainstate.Wallets[tx.Sender].PublicKey)
		if err != nil {
			return B_REJECT_TX_INVALID
		}
		// Verify the transaction
		if !tx.Verify(key) {
			return B_REJECT_TX_INVALID
		}
		// Check that the transaction has the expected id
		if tx.TXID != bc.Chainstate.Wallets[tx.Sender].TXC+1 {
			return B_REJECT_TX_INVALID
		}
		// Check if enough balance exists to make the transaction
		if tx.Amount > bc.Chainstate.Wallets[tx.Sender].Amount {
			return B_REJECT_TX_INVALID
		}
	}
	return B_ACCEPT
}

func (bc *BlockChain) ProcessBlock(b model.Block) error {
	// Process the Registrations in this block
	for _, reg := range b.Registrations {
		bc.Chainstate.Wallets[reg.Wallet] = &WalletInfo{}
		bc.Chainstate.Wallets[reg.Wallet].Amount = 0
		bc.Chainstate.Wallets[reg.Wallet].PublicKey = reg.PublicKey
	}
	// Add the Miners fee to the miners wallet
	bc.Chainstate.Wallets[b.Miner].Amount += model.BlockReward
	bc.Chainstate.MarketVolume += model.BlockReward
	// Process the Transactions
	for _, tx := range b.Transactions {
		bc.Chainstate.Wallets[tx.Sender].Amount -= tx.Amount
		bc.Chainstate.Wallets[tx.Recipient].Amount += tx.Amount
		bc.Chainstate.Wallets[tx.Sender].TXC++
		bc.Chainstate.TransactionVolume++
	}
	// Set the Lastblock to the processed block
	bc.Chainstate.LastBlock = b
	// Append the Block
	bc.Blocks = append(bc.Blocks, &b)
	return nil
}

func VerifyBlock(block model.Block) bool {
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
	bin, err := ioutil.ReadFile("blockchain.json")
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
	err = ioutil.WriteFile("blockchain.json", bin, 0644)
	if err != nil {
		return fmt.Errorf("could not wrtie blockchain with error %v", err)
	}
	return nil
}
