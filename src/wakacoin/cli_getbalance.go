package wakacoin

import (
	"fmt"
	"os"
)

func (cli *CLI) getBalance(address, nodeID string) {
	if ValidateAddress(address) != true {
		errMSG := "ERROR: Address is not valid"
		
		fmt.Println("\n", errMSG)
		os.Exit(1)
	}
	
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()
	
	UTXOSet := UTXOSet{bc}

	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	
	var array [20]byte
	copy(array[:], pubKeyHash)
	balance, balanceSpendable := UTXOSet.Balance(array)

	fmt.Printf("Balance of '%s': %d\n", address, balance)
	fmt.Printf("Spendable Balance of '%s': %d\n", address, balanceSpendable)
}
