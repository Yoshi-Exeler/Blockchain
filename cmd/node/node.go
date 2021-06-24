package main

import (
	"coins/pkg/blockchain"
	"coins/pkg/relay"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
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
	if _, err := os.Stat("blockchain.json"); !os.IsNotExist(err) {
		log.Println("starting to read blockchain")
		chain, err := blockchain.ReadFile()
		if err != nil {
			log.Printf("could not read blockchain with error %v\n", err)
		}
		log.Println("successfully read blockchain file")
		bc = *chain
	}

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

	// Begin Listening for consumers if relaying is active
	if *enableRelay {
		go relay.Listen(":" + *relayPort)
	}

	// Dial our Peers
	go relay.ConsumePeers(peers)

	// Start our miner if it is enabled
	if *enableMiner {
		go relay.MineBlocks(relay.RestartMiner)
	}

	// Block main efficiently
	select {}

}
