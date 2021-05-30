package relay

import (
	"coins/pkg/blockchain"
	"coins/pkg/model"
	"coins/pkg/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type Relay struct {
	Mine         bool
	RestartMiner *bool
	Local        bool
	Connections  []net.Conn
	Blockchain   blockchain.BlockChain
	FloatingTx   []model.Transaction
	FloatingRx   []model.Registration
	Peers        []string
	Wallet       blockchain.Wallet
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
			log.Printf("[MINER] Block found, validating %v  stop:%v wallets:%v\n", newBlock, *stop, r.Blockchain.Chainstate.Wallets)
			if r.Blockchain.ValidateBlock(newBlock) {
				log.Println("[MINER] Block found, broadcasting now")
				// if we found a block, process and broadcast it
				r.BroadcastBlock(newBlock)
				r.Blockchain.ProcessBlock(newBlock)
			}
		}()
	}
}

func (r *Relay) ConsumePeers(peers []string) {
	// Iterate over the specified peers
	for _, peer := range peers {
		// Connect to the peer
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			log.Printf("Could not initialize connection with error %v\n", err)
			continue
		}
		log.Printf("[NODE] Connected to peer %v\n", peer)
		// handle the connection async
		go r.handleConnection(conn)
	}
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
			log.Println("[NODE] Invalid Message Received")
			continue
		}
		// Process the message and respond to it
		r.processAndRespond(msg, conn)
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
	default:
		log.Println("[NODE] Message with invalid type received")
	}
}

func sendMessage(msg protocol.Message, conn net.Conn) {
	bin, _ := json.Marshal(msg)
	fmt.Fprint(conn, string(bin))
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
	// Validate the Block using our current blockchain
	if !r.Blockchain.ValidateBlock(block) {
		log.Println("[NODE] received invalid block, ignoring")
		return
	}
	// Process the Block into our blockchain
	log.Println("[NODE] received new valid block, processing")
	r.Blockchain.ProcessBlock(block)
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
	// Restart our miner
	*r.RestartMiner = true
	// if we are an open relay, broadcast the block
	if !r.Local {
		// Broadcast the block to our peers
		r.BroadcastBlock(block)
	}
}

func remove(s []model.Transaction, i int) []model.Transaction {
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
		r.BroadcastTx(tx)
	}
}

func (r *Relay) handleSync(content string, conn net.Conn) {}

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
