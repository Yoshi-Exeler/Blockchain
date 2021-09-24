package protocol

import (
	"coins/pkg/blockchain"
	"coins/pkg/model"
)

const MESSAGE_DELIMITER = byte(255)

type MessageType byte

const (
	NEW_BLOCK        MessageType = 1
	NEW_TX           MessageType = 2
	SYNC             MessageType = 3
	SYNC_NEXT_BLOCKS MessageType = 4
	INIT             MessageType = 5
	INIT_BLOCKS      MessageType = 6
	NEW_RX           MessageType = 7
)

type Message struct {
	Type    MessageType
	Content string // JSON of the appropriate message
}

type SyncContent struct {
	Head          uint64 // the head block of the remote chainstate
	LastBlockHash string // the hash of the last block
}

type SyncNextBlocksContent struct {
	Head   uint64 // the head of the chainstate of the local relay
	Blocks []*model.Block
}

type InitContent struct {
	SafetyValue byte // how many blocks back to send, more blocks means higher blockchain security but more disk space consumed
}

type InitBlocksContent struct {
	Chainstate *blockchain.Chainstate
	Blocks     []*model.Block
}
