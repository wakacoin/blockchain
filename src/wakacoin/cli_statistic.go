package wakacoin

import (
	"encoding/hex"
	"fmt"
	
	"github.com/boltdb/bolt"
)

func (cli *CLI) statistic(nodeID string) {
	var genesisTime, lastTime int64
	
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()
	
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		
		genesis, err := hex.DecodeString(protectFromGenesis)
		CheckErr(err)
		
		encodedBlock := b.Get(genesis)
		block := DeserializeBlock(encodedBlock)
		genesisTime = block.Header.Timestamp
		
		return nil
	})
	CheckErr(err)
	
	block, err := bc.GetBlock(bc.tip)
	CheckErr(err)

	lastTime = block.Header.Timestamp
	height := block.GetHeight()
	
	averageTime := (lastTime - genesisTime) / int64(height)
	
	fmt.Println("A block is generated every", averageTime, "seconds on average.")
	
	bci := bc.Iterator()
	var emptyArray [32]byte
	var totalMint uint32
	
	for {
		block := bci.Next()
		
		totalMint += block.Transactions[0].Vout[0].Value

		if block.Header.PrevBlock == emptyArray {
			break
		}
	}
	
	fmt.Println("So far, a total of", totalMint, "wakacoins have been minted.")
}
