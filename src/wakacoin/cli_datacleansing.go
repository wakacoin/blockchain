package wakacoin

import (
	"encoding/hex"
	"fmt"
	
	"github.com/boltdb/bolt"
)

func (cli *CLI) dataCleansing(nodeID string) {
	fmt.Println("Data Cleansing")
	
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()
	
	blockHashesFromGenesisToTip = *bc.GetBlockHashes()
	ReverseHashes(blockHashesFromGenesisToTip)
	
	counterI := 0
	counterII := 0
	
	err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		c := b.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			counterI++
			blockIsUseful := false
			block := DeserializeBlock(v)
			height := block.GetHeight()
			
			switch {
			case height > protectHeight:
				blockIsUseful = true
				
			case height == protectHeight:
				var blockHash [32]byte
				copy(blockHash[:], k)
				
				slice, err := hex.DecodeString(protectTo)
				CheckErr(err)
				
				var protectToArray [32]byte
				copy(protectToArray[:], slice)
				
				if blockHash == protectToArray {
					blockIsUseful = true
				}
				
			case height == 0:
				var blockHash [32]byte
				copy(blockHash[:], k)
				
				slice, err := hex.DecodeString(protectFromGenesis)
				CheckErr(err)
				
				var protectFromGenesisArray [32]byte
				copy(protectFromGenesisArray[:], slice)
				
				if blockHash == protectFromGenesisArray {
					blockIsUseful = true
				}
				
			default:
				var blockHash [32]byte
				copy(blockHash[:], k)
				
				for _, ID := range blockHashesFromGenesisToTip {
					if blockHash == ID {
						blockIsUseful = true
					}
				}
			}
			
			if blockIsUseful == false {
				UTXOSet := UTXOSet{bc}
				
				for index, tx := range block.Transactions {
					if index != 0 {
						isSpendable, err := UTXOSet.isSpendableTX(tx)
						
						isOK := false
						
						if isSpendable {
							isOK = true
							
						} else {
							if err.Error() == confirmationsError {
								isOK = true
							}
						}
						
						if isOK {
							_ = tx.AddNewTxToBucket(bc.db)
						}
					}
				}
				
				err := b.Delete(k)
				CheckErr(err)
				
				counterII++
			}
		}
		
		verifyB := tx.Bucket([]byte(verifyBucket))
		c = verifyB.Cursor()
		
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			blk := b.Get(k)
			
			if blk == nil {
				err := verifyB.Delete(k)
				CheckErr(err)
				
			} else {
				block := DeserializeBlock(blk)
				height := block.GetHeight()
				
				if height <= protectHeight {
					err := verifyB.Delete(k)
					CheckErr(err)
				}
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	fmt.Printf("There are %d blocks in the database.\n", counterI)
	fmt.Printf("The %d block(s) have been deleted.\n", counterII)
	
	fmt.Println("\n")
}