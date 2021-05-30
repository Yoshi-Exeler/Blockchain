package main

import (
	"flag"
	"fmt"
	"os"
)

const MaxBlocksPerRequest = 100

func main() {
	enableRelay := flag.Bool("relay-enable", false, "Whether or not to enable relaying on the relay port")
	relayPort := flag.String("relay-port", "10505", "The port used to relay messages to other nodes")
	peerFile := flag.String("peer-file", "peers.json", "Path to the file containing peer nodes to establish connections with")
	showHelp := flag.Bool("help", false, "Shows this Help page")

	flag.Parse()

	// Display help page
	if *showHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	fmt.Println(*enableRelay, *relayPort, *peerFile)

}
