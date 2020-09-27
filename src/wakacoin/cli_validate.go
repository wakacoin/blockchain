package wakacoin

import (
	"fmt"
	"os"
	
	"github.com/boltdb/bolt"
)

func (cli *CLI) validate(nodeID string) {
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()
	
	blockHashesFromGenesisToTip = *bc.GetBlockHashes()
	ReverseHashes(blockHashesFromGenesisToTip)
	
	fmt.Println("Verify Protect")
	
	err := bc.VerifyProtect()
	
	if err != nil {
		fmt.Println("\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Verify Blocks")
	
	bci := bc.Iterator()
	var emptyArray [32]byte
	
	block, err := bc.GetBlock(bc.tip)
	CheckErr(err)
	
	height := block.GetHeight()
	counter := 0
	
	for {
		counter++
		block := bci.Next()
		
		if err := VerifyBlockWithoutChain(block); err != nil {
			fmt.Println("\n", "BlockHeight ", height, ": ", err)
			
			err := bc.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(paramBucket))
				err := b.Put([]byte("valid"), []byte("N"))
				CheckErr(err)
				
				return nil
			})
			CheckErr(err)
			
			os.Exit(1)
		}
		
		if err := bc.VerifyBlock(block, &blockHashesFromGenesisToTip); err != nil {
			fmt.Println("\n", "BlockHeight ", height, ": ", err)
			
			err := bc.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(paramBucket))
				err := b.Put([]byte("valid"), []byte("N"))
				CheckErr(err)
				
				return nil
			})
			CheckErr(err)
			
			os.Exit(1)
		}
		
		if block.Header.PrevBlock == emptyArray {
			err := bc.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(paramBucket))
				err := b.Put([]byte("valid"), []byte("Y"))
				CheckErr(err)
				
				return nil
			})
			CheckErr(err)
			
			break
		}
		
		height--
		
		if counter % 1000 == 0 {
			fmt.Printf("\n\n\n%d blocks\n\n\n", counter)
		}
	}
	
	fmt.Println("\n")
	
	fmt.Printf("There are %d blocks in the blockchain.\n", counter)
	fmt.Printf("The %d blocks have been validated.\n", counter)
	
	fmt.Println("\n")
}