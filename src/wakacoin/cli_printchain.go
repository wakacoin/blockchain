package wakacoin

import (
	"fmt"
	"time"
)

func (cli *CLI) printChain(page uint, nodeID string) {
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()

	bci := bc.Iterator()
	var emptyArray [32]byte
	
	x := int(page) * 2
	y := x - 3
	for i := 0; i < x; i++ {
		blockHash := bci.currentHash
		block := bci.Next()

		if i > y {
			fmt.Printf("============ Block %x ============\n", blockHash)
			
			fmt.Printf("Height:     %d\n", block.GetHeight())
			
			fmt.Printf("Version:    %d\n", block.Header.Version)
			fmt.Printf("PrevBlock:  %x\n", block.Header.PrevBlock)
			fmt.Printf("MerkleRoot: %x\n", block.Header.MerkleRoot)
			fmt.Printf("Timestamp:  %d\n", block.Header.Timestamp)
			fmt.Printf("Time:       %s\n", time.Unix(block.Header.Timestamp, 0).Format("2006-01-02 15:04:05"))
			fmt.Printf("Difficulty: %d\n", block.Header.Difficulty)
			fmt.Printf("Nonce:      %d\n", block.Header.Nonce)
			
			fmt.Printf("\n")
			
			for _, tx := range block.Transactions {
				fmt.Println(tx)
			}
			
			fmt.Printf("\n\n")
		}

		if block.Header.PrevBlock == emptyArray {
			break
		}
	}
}
