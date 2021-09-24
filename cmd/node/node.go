package main

import (
	"coins/pkg/blockchain"
	"coins/pkg/model"
	"coins/pkg/relay"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

const MaxBlocksPerRequest = 100

func main() {
	enableRelay := flag.Bool("relay-enable", false, "Whether or not to enable relaying on the relay port")
	relayPort := flag.String("relay-port", "10505", "The port used to relay messages to other nodes")
	peerFile := flag.String("peer-file", "peers.json", "Path to the file containing peer nodes to establish connections with")
	enableMiner := flag.Bool("miner-enable", false, "Whether or not to mine coins")
	showHelp := flag.Bool("help", false, "Shows this Help page")

	flag.Parse()

	// Display help page
	if *showHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Parse the Wallet file
	var wallet *blockchain.Wallet
	wallet, err := blockchain.ReadWalletFile()
	if err != nil {
		// if parsing fails, generate a new wallet instead
		log.Printf("Failed to read wallet file with error %v\n", err)
		wallet, err = blockchain.GenerateWalletFile()
		if err != nil {
			log.Printf("could not generate wallet with error %v\n", err)
		}
	}

	// Parse the blockchain file
	var bc blockchain.BlockChain
	// if a blockchain.json exists, parse it
	log.Println("starting to read blockchain")
	chain, err := blockchain.ReadFile()
	if err != nil {
		log.Printf("could not read blockchain with error %v now initializing\n", err)
		chain = &blockchain.BlockChain{
			Blocks: []*model.Block{},
			Chainstate: blockchain.Chainstate{
				Wallets: make(map[string]*blockchain.WalletInfo),
			},
		}
	}
	log.Println("successfully read blockchain file")
	bc = *chain

	// Parse the peer file
	content, err := ioutil.ReadFile(*peerFile)
	if err != nil {
		log.Fatalf("could not read peer file with error %v\n", err)
	}
	var peers []string
	err = json.Unmarshal(content, &peers)
	if err != nil {
		log.Fatalf("could not unmarshall peer file with error %v\n", err)
	}

	restart := false

	// Create our Relay
	relay := relay.Relay{
		Local:         *enableRelay,
		Blockchain:    bc,
		Peers:         peers,
		Wallet:        *wallet,
		RestartMiner:  &restart,
		PeerSyncMutex: &sync.Mutex{},
	}

	// Make sure we register with the blockchain
	relay.RgisterOrNop()

	// Begin Listening for consumers if relaying is active
	if *enableRelay {
		go relay.Listen(":" + *relayPort)
	}

	// Dial our Peers
	go relay.ConsumePeers(peers)

	// If we are not registered, we must register
	if relay.Blockchain.Chainstate.Wallets[relay.Wallet.Address] == nil {
		kstr, err := blockchain.KeyToString(&relay.Wallet.KP.PublicKey)
		if err != nil {
			log.Fatal("unable to convert public key to string")
		}
		rx := model.Registration{
			Wallet:    relay.Wallet.Address,
			PublicKey: kstr,
		}
		relay.FloatingRx = append(relay.FloatingRx, rx)
		relay.BroadcastRx(rx)
	}

	// Start our miner if it is enabled
	if *enableMiner {
		go relay.MineBlocks(relay.RestartMiner)
	}

	// Make sure we regularly commit the blockchain to disk
	go relay.CommitBlockchain()

	// Read stdin and process commands
	for {
		inputBuffer, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("[NODECTL] error parsing command")
		}
		// Split the command into its parts
		parts := strings.Split(string(inputBuffer), " ")
		// Switch the command type
		switch parts[0] {
		case "send":
		case "":
		}
	}
	// Block main efficiently
	select {}
}
