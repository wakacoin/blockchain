package wakacoin

import (
	"encoding/hex"
	"fmt"
)

func (cli *CLI) prinTransaction(blockid, txid string, nodeID string) {
	var blockID, txID [32]byte
	
	decoded, err := hex.DecodeString(blockid)
	CheckErr(err)
	
	copy(blockID[:], decoded)
	
	decoded, err = hex.DecodeString(txid)
	CheckErr(err)
	
	copy(txID[:], decoded)
	
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()

	bci := bc.Iterator()
	var emptyArray [32]byte
	
	for {
		blockHash := bci.currentHash
		block := bci.Next()
		
		if blockHash == blockID {
			for _, tx := range block.Transactions {
				if tx.ID == txID {
					fmt.Printf("============ Block %x ============\n", blockHash)
					fmt.Printf("Height:     %d\n", block.GetHeight())
					
					fmt.Println(tx)
					
					return
				}
			}
		}

		if block.Header.PrevBlock == emptyArray {
			fmt.Println("The transaction is not found")
			
			return
		}
	}
}
