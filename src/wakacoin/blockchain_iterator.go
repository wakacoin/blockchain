package wakacoin

import (
	"github.com/boltdb/bolt"
)

type BlockchainIterator struct {
	currentHash [32]byte
	db          *bolt.DB
}

type BlockchainIteratorTwo struct {
	currentHash [32]byte
	blockHeight uint32
	db          *bolt.DB
}

func (i *BlockchainIterator) Next() *Block {
	var block *Block
	
	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash[:])
		block = DeserializeBlock(encodedBlock)

		return nil
	})
	
	CheckErr(err)
	
	i.currentHash = block.Header.PrevBlock
	
	return block
}

func (i *BlockchainIteratorTwo) NextIfBlockExists() (findNonExistingBlock, findIllegalBlock bool, blockHash [32]byte, height uint32) {
	var block *Block
	var encodedBlock []byte
	
	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock = b.Get(i.currentHash[:])
		
		if encodedBlock != nil {
			block = DeserializeBlock(encodedBlock)
		}

		return nil
	})
	CheckErr(err)
	
	if encodedBlock == nil {
		return true, false, i.currentHash, i.blockHeight
		
	} else {
		if block.GetHeight() != i.blockHeight {
			return false, true, i.currentHash, i.blockHeight	
		}
		
		err = VerifyBlockWithoutChain(block)
		
		if err != nil {
			return false, true, i.currentHash, i.blockHeight
			
		} else {
			i.currentHash = block.Header.PrevBlock
			i.blockHeight--
			return false, false, i.currentHash, i.blockHeight
		}		
	}
}