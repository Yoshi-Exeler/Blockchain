package relay

import (
	"coins/pkg/blockchain"
	"coins/pkg/gorx"
	"coins/pkg/model"
	"coins/pkg/protocol"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
)

type Relay struct {
	Mine          bool
	RestartMiner  *bool
	Local         bool
	Connections   []net.Conn
	Blockchain    blockchain.BlockChain
	FloatingTx    []model.Transaction
	FloatingRx    []model.Registration
	Peers         []string
	Wallet        blockchain.Wallet
	PeerSyncMutex *sync.Mutex
	SyncPromise   *gorx.Promise
	InSyncTx      bool
}

func (r *Relay) MineBlocks(stop *bool) {
	for {
		// Create our new block
		newBlock := model.Block{
			ID:            r.Blockchain.Chainstate.LastBlock.ID + 1,
			Nonce:         0,
			Previous:      r.Blockchain.Chainstate.LastBlock.Hash,
			Miner:         r.Wallet.Address,
			Transactions:  r.FloatingTx,
			Registrations: r.FloatingRx,
		}
		// restart
		*stop = false
		// Launch miner
		go func() {
			newBlock.Mine(stop)
			// if the miner stops, check if its result is a valid block
			res := r.Blockchain.ValidateBlock(newBlock)
			if res == blockchain.B_ACCEPT {
				log.Printf("[MINER] new block mined id=%v hash=%v\n", newBlock.ID, newBlock.Hash)
				// if we found a block, process and broadcast it
				r.newBlock(newBlock)
			}
		}()
		for {
			time.Sleep(time.Millisecond * 100)
			if *stop {
				break
			}
		}
	}
}

func (r *Relay) CommitBlockchain() {
	for {
		// copy the blockchain
		state := r.Blockchain
		// marhsall the copy
		bin, err := json.Marshal(state)
		if err != nil {
			fmt.Printf("[NODE] failed to marshall blockchain with error %v\n", err)
		}
		// Write the content to file
		err = ioutil.WriteFile("blockchain.json", bin, 0644)
		if err != nil {
			fmt.Printf("[NODE] failed to commit blockchain to disk with error %v\n", err)
		}
		fmt.Println("[NODE] blockchain committed to disk successfully")
		time.Sleep(time.Second * 10)
	}

}

func (r *Relay) RegisterOrNop() {
	// First we need to check if we are registered on the blockchain
	if r.Blockchain.Chainstate.Wallets[r.Wallet.Address] != nil {
		// If we are registered just nop
		return
	}
	// Convert our private key
	keyStr, err := blockchain.KeyToString(&r.Wallet.KP.PublicKey)
	if err != nil {
		fmt.Println("[NODE] blockchain registration request building failed")
		return
	}
	// Build a registration
	rx := model.Registration{
		Wallet:    r.Wallet.Address,
		PublicKey: keyStr,
	}
	// Broadcast onto the network
	go r.BroadcastRx(rx)
	// Add it to our own floating rx
	r.FloatingRx = append(r.FloatingRx, rx)
}

func (r *Relay) ConsumePeers(peers []string) {
	// Iterate over the specified peers
	for _, peer := range peers {
		alloc := peer
		go func() {
			for {
				// Connect to the peer
				conn, err := net.Dial("tcp", alloc)
				if err != nil {
					log.Printf("Could not initialize connection with error %v\n", err)
					continue
				}
				log.Printf("[NODE] Connected to peer %v\n", alloc)
				// handle the connection async
				r.handleConnection(conn)
				time.Sleep(time.Second * 1)
			}
		}()
	}
}

func (r *Relay) TrySyncOrNop(conn net.Conn) {
	// Check if we are currently in a sync transaction
	if r.InSyncTx {
		// if a sync transaction is currently active, just nop
		return
	}
	// If we are not in a sync transaction, try to acquire the sync lock
	r.PeerSyncMutex.Lock()
	// Now we can begin syncing with a peer, we will use the peer specified
	// Build a message content string
	cont := protocol.SyncContent{LastBlockHash: r.Blockchain.Chainstate.LastBlock.Hash}
	// Marshall the content to json
	bin, err := json.Marshal(cont)
	if err != nil {
		fmt.Println("[RELAY] failed to build sync request")
	}
	// Build a message
	msg := protocol.Message{Type: protocol.SYNC, Content: string(bin)}
	// Setup our sync promise
	r.SyncPromise = gorx.NewPromiseWithTimeout(time.Minute).Then(func(v interface{}) {
		r.SyncPromise = nil
		r.PeerSyncMutex.Unlock()
		r.InSyncTx = false
	})
	// Log
	fmt.Println("[RELAY] syncing with peer ", conn.RemoteAddr())
	// Send the sync request to the node
	sendMessage(msg, conn)
}

func (r *Relay) Listen(addr string) {
	log.Printf("now listening for consumers on %v\n", addr)
	// Initialize a socket to accept tcp connections
	soc, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Could not initialize socket with error %v\n", err)
	}
	// Accept all incomming connections and handle them in separate go routines
	for {
		// Wait for the next connection and accept it
		connection, err := soc.Accept()
		if err != nil {
			log.Fatalf("Could not accept connection with error %v\n", err)
		}
		log.Printf("[NODE] Accepted Consumer %v\n", connection.RemoteAddr())
		// add the consumer to the broadcast pool
		r.Connections = append(r.Connections, connection)
		// handle the connection async
		go r.handleConnection(connection)
	}
}

// handleConnection handles the communication with a connection
func (r *Relay) handleConnection(conn net.Conn) {
	// Begin Reading the connection
	decoder := json.NewDecoder(conn)
	for {
		// Read until a full message is received
		var msg protocol.Message
		err := decoder.Decode(&msg)
		if err != nil {
			if &msg == nil {
				return
			}
			log.Println("[NODE] Invalid Message Received")
			continue
		}
		// Process the message and respond to it
		go r.processAndRespond(msg, conn)
	}

}

// processAndRespond calls the appropriate message handler depending on the message type
func (r *Relay) processAndRespond(msg protocol.Message, conn net.Conn) {
	switch msg.Type {
	case protocol.NEW_BLOCK:
		r.handleNewBlock(msg.Content, conn)
	case protocol.NEW_TX:
		r.handleNewTX(msg.Content, conn)
	case protocol.SYNC:
		r.handleSync(msg.Content, conn)
	case protocol.INIT:
		r.handleInit(msg.Content, conn)
	case protocol.SYNC_NEXT_BLOCKS:
		r.handleSyncNextBlocks(msg.Content, conn)
	case protocol.NEW_RX:
		r.handleNewRx(msg.Content, conn)
	default:
		log.Println("[NODE] Message with invalid type received")
	}
}

func (r *Relay) handleNewRx(content string, conn net.Conn) {
	// Unmarshall the content
	var req model.Registration
	err := json.Unmarshal([]byte(content), &req)
	if err != nil {
		log.Println("[NODE] failed to unmarshall blocks")
		return
	}
	// Log that we received a new rx
	fmt.Printf("[NODE] Received new Registration for %v\n", req.Wallet)
	// Add the registration to the pool of floating rx
	r.FloatingRx = append(r.FloatingRx, req)
}

func (r *Relay) handleSyncNextBlocks(content string, conn net.Conn) {
	// Unmarshall the content
	var req protocol.SyncNextBlocksContent
	err := json.Unmarshal([]byte(content), &req)
	if err != nil {
		log.Println("[NODE] failed to unmarshall blocks")
		return
	}
	// check if the remote state has more work than ours
	if req.Head <= r.Blockchain.Chainstate.LastBlock.ID {
		fmt.Printf("[NODE] Sync from peer %v rejected. remote head %v <= local head %v", conn.RemoteAddr(), req.Head, r.Blockchain.Chainstate.LastBlock.ID)
		return
	}
	// Write to log
	fmt.Printf("[NODE] Received %v blocks from peer %v\n", len(req.Blocks), conn.RemoteAddr())
	// Process the blocks in reverse order
	for i := len(req.Blocks) - 1; i >= 0; i-- {
		fmt.Printf("BEGIN_PROCESS_BLOCK %v\n", req.Blocks[i].ID)
		r.newBlockFromPeer(*req.Blocks[i], conn)
	}
	// Complete our sync promise
	r.SyncPromise.Resolve(nil)
}

func sendMessage(msg protocol.Message, conn net.Conn) {
	bin, _ := json.Marshal(msg)
	fmt.Fprint(conn, string(bin))
}

func (r *Relay) newBlock(block model.Block) {
	// Process the Block into our blockchain
	log.Printf("[NODE] new block id=%v accepted\n", block.ID)
	go r.Blockchain.ProcessBlock(block)
	// Remove its transactions from the floating ones
	actionTaken := false
	for {
		for i := 0; i < len(block.Transactions); i++ {
			for j := 0; j < len(r.FloatingTx); j++ {
				r.FloatingTx = remove(r.FloatingTx, j)
				actionTaken = true
				break
			}
			if actionTaken {
				break
			}
		}
		if !actionTaken {
			break
		}
	}
	actionTaken = false
	for {
		for i := 0; i < len(block.Registrations); i++ {
			for j := 0; j < len(r.FloatingRx); j++ {
				r.FloatingRx = removeRX(r.FloatingRx, j)
				actionTaken = true
				break
			}
			if actionTaken {
				break
			}
		}
		if !actionTaken {
			break
		}
	}
	// Restart our miner
	*r.RestartMiner = true
	// if we are an open relay, broadcast the block
	if !r.Local {
		// Broadcast the block to our peers
		go r.BroadcastBlock(block)
	}
}

func (r *Relay) newBlockFromPeer(block model.Block, conn net.Conn) {
	// Validate the Block using our current blockchain
	res := r.Blockchain.ValidateBlock(block)
	if res != blockchain.B_ACCEPT {
		log.Printf("[NODE] block with id=%v rejected with reason=%v\n", block.ID, res)
		go r.TrySyncOrNop(conn)
		return
	}
	r.newBlock(block)
}

func (r *Relay) handleNewBlock(content string, conn net.Conn) {
	log.Printf("[%v->%v] New Block", conn.RemoteAddr(), conn.LocalAddr())
	// Unmarshall the message content
	var block model.Block
	err := json.Unmarshal([]byte(content), &block)
	if err != nil {
		log.Println("[NODE] Failed to unmarshall new block, ignoring")
		return
	}
	r.newBlockFromPeer(block, conn)
}

func remove(s []model.Transaction, i int) []model.Transaction {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func removeRX(s []model.Registration, i int) []model.Registration {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func (r *Relay) handleNewTX(content string, conn net.Conn) {
	log.Printf("[%v->%v] New Transaction", conn.RemoteAddr(), conn.LocalAddr())
	// Unmarshall the message content
	var tx model.Transaction
	err := json.Unmarshal([]byte(content), &tx)
	if err != nil {
		log.Println("[NODE] Failed to unmarshall new transaction, ignoring")
		return
	}
	// Get the Public key of the supposed sender of the transaction
	key, err := blockchain.StringToKey(r.Blockchain.Chainstate.Wallets[tx.Sender].PublicKey)
	if err != nil {
		log.Println("[NODE] unknown transaction sender, ignoring")
		return
	}
	// Validate the Transaction signature
	if !tx.Verify(key) {
		log.Println("[NODE] transaction signature is not valid, ignoring")
		return
	}
	// Add the transaction to the floating transactions
	r.FloatingTx = append(r.FloatingTx, tx)
	// if we are an open relay, broadcast the transaction
	if !r.Local {
		// Broadcast the block to our peers
		go r.BroadcastTx(tx)
	}
}

func (r *Relay) handleSync(content string, conn net.Conn) {
	log.Printf("[%v->%v] Sync request", conn.RemoteAddr(), conn.LocalAddr())
	// Unmarshall the message content
	var syncHeader protocol.SyncContent
	err := json.Unmarshal([]byte(content), &syncHeader)
	if err != nil {
		log.Println("[NODE] Failed to unmarshall sync header")
		return
	}
	// Find the blocks that the other node is missing
	missingBlocks := []*model.Block{}
	for i := len(r.Blockchain.Blocks) - 1; i >= 0; i-- {
		if r.Blockchain.Blocks[i].Hash == syncHeader.LastBlockHash {
			break
		}
		missingBlocks = append(missingBlocks, r.Blockchain.Blocks[i])
	}
	response := protocol.SyncNextBlocksContent{Blocks: missingBlocks}
	// Marshall the response
	bin, err := json.Marshal(response)
	if err != nil {
		log.Println("[NODE] Failed to marshall sync response content")
		return
	}
	// Create the Message
	msg := protocol.Message{
		Type:    protocol.SYNC_NEXT_BLOCKS,
		Content: string(bin),
	}
	// Marshall the message
	msgBin, err := json.Marshal(msg)
	if err != nil {
		log.Println("[NODE] Failed to marshall message")
		return
	}
	// Respond with the message
	fmt.Fprint(conn, string(msgBin))
}

func (r *Relay) handleInit(content string, conn net.Conn) {}

func (r *Relay) BroadcastBlock(block model.Block) {
	// Marhsall the block
	bin, err := json.Marshal(block)
	if err != nil {
		log.Println("[NODE] Could not broadcast block because serialization of the block failed")
		return
	}
	// Create our broadcast message
	msg := protocol.Message{Type: protocol.NEW_BLOCK, Content: string(bin)}
	// Marshall the message
	msgBuffer, err := json.Marshal(msg)
	if err != nil {
		log.Println("[NODE] Could not broadcast block because serialization of the message failed")
		return
	}
	// Send the marshalled message to each connected consumer
	for _, conn := range r.Connections {
		log.Printf("[%v->%v] BLOCK:%v", conn.LocalAddr(), conn.RemoteAddr(), block.Hash)
		fmt.Fprint(conn, string(msgBuffer))
	}
}

func (r *Relay) BroadcastTx(tx model.Transaction) {
	// Marhsall the transaction
	bin, err := json.Marshal(tx)
	if err != nil {
		log.Println("[NODE] Could not broadcast transaction because serialization of the transaction failed")
		return
	}
	// Create our broadcast message
	msg := protocol.Message{Type: protocol.NEW_TX, Content: string(bin)}
	// Marshall the message
	msgBuffer, err := json.Marshal(msg)
	if err != nil {
		log.Println("[NODE] Could not broadcast transaction because serialization of the message failed")
		return
	}
	// Send the marshalled message to each connected consumer
	for _, conn := range r.Connections {
		log.Printf("[%v->%v] TX:%v", conn.LocalAddr(), conn.RemoteAddr(), tx.Hash)
		fmt.Fprint(conn, string(msgBuffer))
	}
}

func (r *Relay) BroadcastRx(rx model.Registration) {
	// Marhsall the Registration
	bin, err := json.Marshal(rx)
	if err != nil {
		log.Println("[NODE] Could not broadcast registration because serialization of the registration failed")
		return
	}
	// Create our broadcast message
	msg := protocol.Message{Type: protocol.NEW_RX, Content: string(bin)}
	// Marshall the message
	msgBuffer, err := json.Marshal(msg)
	if err != nil {
		log.Println("[NODE] Could not broadcast registration because serialization of the message failed")
		return
	}
	// Send the marshalled message to each connected consumer
	for _, conn := range r.Connections {
		log.Printf("[%v->%v] RX:%v", conn.LocalAddr(), conn.RemoteAddr(), rx.Wallet)
		fmt.Fprint(conn, string(msgBuffer))
	}
}
