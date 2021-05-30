package relay

import (
	"coins/pkg/model"
	"coins/pkg/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type Relay struct {
	Connections []net.Conn
}

func (r *Relay) Listen(addr string) {
	// Initialize a socket to accept tcp connections
	soc, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Could not initialize socket with error %v", err)
	}
	// Accept all incomming connections and handle them in separate go routines
	for {
		// Wait for the next connection and accept it
		connection, err := soc.Accept()
		if err != nil {
			log.Fatalf("Could not accept connection with error %v", err)
		}
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

func (r *Relay) handleNewBlock(content string, conn net.Conn) {}

func (r *Relay) handleNewTX(content string, conn net.Conn) {}

func (r *Relay) handleSync(content string, conn net.Conn) {}

func (r *Relay) handleInit(content string, conn net.Conn) {}

func (r *Relay) BroadcastBlock(block model.Block) {
	// Marhsall the block
	bin, err := json.Marshal(block)
	if err != nil {
		log.Print("[NODE] Could not broadcast block because serialization of the block failed")
	}
	// Create our broadcast message
	msg := protocol.Message{Type: protocol.NEW_BLOCK, Content: string(bin)}
	// Marshall the message
	msgBuffer, err := json.Marshal(msg)
	if err != nil {
		log.Print("[NODE] Could not broadcast block because serialization of the message failed")
	}
	// Send the marshalled message to each connected consumer
	for _, conn := range r.Connections {
		fmt.Fprint(conn, string(msgBuffer))
	}
}

func (r *Relay) BroadcastTx(tx model.Transaction) {
	// Marhsall the transaction
	bin, err := json.Marshal(tx)
	if err != nil {
		log.Print("[NODE] Could not broadcast transaction because serialization of the transaction failed")
	}
	// Create our broadcast message
	msg := protocol.Message{Type: protocol.NEW_TX, Content: string(bin)}
	// Marshall the message
	msgBuffer, err := json.Marshal(msg)
	if err != nil {
		log.Print("[NODE] Could not broadcast transaction because serialization of the message failed")
	}
	// Send the marshalled message to each connected consumer
	for _, conn := range r.Connections {
		fmt.Fprint(conn, string(msgBuffer))
	}
}
