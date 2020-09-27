package wakacoin

import (
	"fmt"
	"os"
	
	"github.com/boltdb/bolt"
)

func (cli *CLI) prinTxsBucket(nodeID string) {
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()
	
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(txsBucket))
		c := b.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			tx := DeserializeTransaction(v)
			
			fmt.Println(&tx)
			fmt.Println("\n")
			
			txSize := len(v)
			
			fmt.Println("tx size: ", txSize , "bytes\n")
		}
		
		return nil
	})
	CheckErr(err)
	
	os.Exit(1)
}