package protocol

import (
	"coins/pkg/blockchain"
	"coins/pkg/model"
)

type MessageType byte

const (
	NEW_BLOCK        MessageType = 1
	BLOCK_OK         MessageType = 2
	BLOCK_REJECTED   MessageType = 3
	NEW_TX           MessageType = 4
	TX_OK            MessageType = 5
	TX_REJECTED      MessageType = 6
	SYNC             MessageType = 7
	SYNC_STATE_OK    MessageType = 8
	SYNC_NEXT_BLOCKS MessageType = 9
	SNYC_BC_INVALID  MessageType = 10
	INIT             MessageType = 11
	INIT_BLOCKS      MessageType = 12
)

type Message struct {
	Type    MessageType
	Content string // JSON of the appropriate message
}

type SyncContent struct {
	LastBlockHash string
}

type SyncNextBlocksContent struct {
	Blocks []*model.Block
}

type InitContent struct {
	SafetyValue byte // how many blocks back to send, more blocks means higher blockchain security but more disk space consumed
}

type InitBlocksContent struct {
	Chainstate *blockchain.Chainstate
	Blocks     []*model.Block
}
