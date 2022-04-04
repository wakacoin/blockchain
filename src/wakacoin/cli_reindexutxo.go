package wakacoin

import "fmt"

func (cli *CLI) reindexUTXO(nodeID string) {
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()
	
	blockHashesFromGenesisToTip = *bc.GetBlockHashes()
	ReverseHashes(blockHashesFromGenesisToTip)
	
	UTXOSet := UTXOSet{bc}
	
	for i := 0; i < 2; i++ {
		utxoBucket, contractBucket := UTXOSet.GetUnAvailableUTXO()
		
		UTXOSet.ResetUTXOLastBlock(utxoBucket)
		
		err := UTXOSet.Rebuild(utxoBucket, contractBucket)
		CheckErr(err)
		
		UTXOSet.SetAvailable(utxoBucket, bc.tip)
	}
	
	fmt.Println("Done")
}
