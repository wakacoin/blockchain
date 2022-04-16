package wakacoin

import (
	"fmt"
	"os"
	"time"
)

func (cli *CLI) startNode(nodeID, minerAddress, from, to string, sendNewTx bool, amount uint32) {
	fmt.Printf("Starting node %s\n", nodeID)
	
	if len(minerAddress) > 0 {
		if ValidateAddress(minerAddress) {
			fmt.Println(time.Now().UTC())
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
			
		} else {
			PrintMessage("ERROR: Wrong miner address")
			os.Exit(1)
		}
	}
	
	StartServer(nodeID, minerAddress, from, to, sendNewTx, amount)
}
