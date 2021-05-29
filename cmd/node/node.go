package main

import (
	"flag"
	"fmt"
	"os"
)

// the job of a node is to propagate transactions to miners and blocks to everyone on the network

// Register for Block Broadcasts at

func main() {
	blockCastPort := flag.String("block-broadcast-port", "10545", "The port on which new block will be broadcast")
	txCastPort := flag.String("transaction-broadcast-port", "10546", "The port on which pending transactions will be broadcast")
	txSubmitPort := flag.String("transaction-submit-port", "10547", "The port on which to submit new transactions")
	blockSubmitPort := flag.String("block-submit-port", "10548", "The port on which to submit new blocks")
	showHelp := flag.Bool("help", false, "Shows this Help page")

	flag.Parse()

	if *showHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	fmt.Println(*blockCastPort, *txCastPort, *txSubmitPort, *blockSubmitPort)
}
