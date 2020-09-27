package wakacoin

import (
	"fmt"
)

func (cli *CLI) listAddresses(nodeID string) {
	wallets, err := NewWallets(nodeID)
	CheckErr(err)
	
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}
