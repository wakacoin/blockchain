package wakacoin

import (
	"fmt"
	"os"
	"time"
	
	"github.com/boltdb/bolt"
)

func (cli *CLI) sendTXs(nodeID string) {
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()
	
	nodeId = nodeID
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	hostStaticAddress = LocalhostStaticIPAddr + ":" + nodeID
	hostDomainName = LocalhostDomainName + ":" + nodeID
	
	switch {
	case nodeAddress == DefaultHub:
		localhostisDefaultHub = true
		
	case SetLocalhostDomainName && hostDomainName == DefaultHub:
		localhostisDefaultHub = true
		
	case SetLocalhostStaticIPAddr && hostStaticAddress == DefaultHub:
		localhostisDefaultHub = true
	}
	
	if localhostisDefaultHub {
		errMSG := "ERROR: localhost is the DefaultHub."
		
		fmt.Println("\n", errMSG)
		os.Exit(1)
	}
	
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(txsBucket))
		c := b.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			sendTx(DefaultHub, v, bc.db, true)
		}

		return nil
	})
	CheckErr(err)
	
	time.Sleep(30 * time.Second)
	os.Exit(1)
}