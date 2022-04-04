package wakacoin

import (
	"fmt"
	"os"
	"time"
	
	"github.com/boltdb/bolt"
)

func (cli *CLI) tool(nodeID string) {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	
	if dbExists(dbFile) == false {
		errMSG := "ERROR: Require the database file - Wakacoin_1024.db. Please download the latest version of the database from https://each1.net/public/wakacoin/"
		
		fmt.Println("\n", errMSG)
		os.Exit(1)
	}
	
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	CheckErr(err)

	defer db.Close()
	
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(contractBucket_A))
		CheckErr(err)

		_, err = tx.CreateBucketIfNotExists([]byte(contractBucket_B))
		CheckErr(err)
	
		return nil
	})
	CheckErr(err)
	
	fmt.Println("Done")
}