package wakacoin

import (
	"bytes"
	"encoding/gob"
	"errors"
	
	"github.com/boltdb/bolt"
)

type Block struct {
	Header       *BlockHeader
	Transactions []*Transaction
}

func (b *Block) GetHeight() uint32 {
	return ByteToUint32(b.Transactions[0].Vin[0].PubKey)
}

func (b *Block) AddBlock(db *bolt.DB) {
	blockHash := b.Header.HashBlockHeader()
	
	err := db.Update(func(tx *bolt.Tx) error {
		blocks := tx.Bucket([]byte(blocksBucket))
		
		blockExists := blocks.Get(blockHash[:])
		
		if blockExists == nil {
			err := blocks.Put(blockHash[:], b.Serialize())
			CheckErr(err)
		}
		
		return nil
	})
	CheckErr(err)
}

func (b *Block) HashTransactions() [32]byte {
	var txids [][32]byte
	
	for _, tx := range b.Transactions {
		txids = append(txids, tx.ID)
	}
	
	mTree := NewMerkleTree(txids)
	
	return mTree
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	CheckErr(err)

	return result.Bytes()
}

func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	CheckErr(err)

	return &block
}

func NewBlock(prevBlock [32]byte, transactions []*Transaction, difficulty uint8, bc *Blockchain, batch uint32, maxNonce uint64, sleep uint8) (*Block, error) {	
	block := &Block{}
	block.Transactions = transactions
	txAmount := len(transactions)
	var merkleRoot [32]byte
	
	if txAmount < 1 {
		return block, errors.New("Coinbase is not found.")
	}
	
	if block.Transactions[0].IsCoinbase() != true {
		return block, errors.New("Coinbase must be the first transaction.")
	}
	
	if txAmount == 1 {
		merkleRoot = block.Transactions[0].ID
		
	} else if txAmount > 1 {
		merkleRoot = block.HashTransactions()
	}
	
	blockHeader, err := NewBlockHeader(prevBlock, merkleRoot, difficulty, bc, batch, maxNonce, sleep)
	
	if err == nil {
		block.Header = blockHeader
		return block, nil
	}
	
	return block, err
}

func NewGenesisBlock(coinbase *Transaction, bc *Blockchain) *Block {
	block, err := NewBlock([32]byte{}, []*Transaction{coinbase}, difficultyDefault_0, bc, 4294967295, 0, 0)	
	CheckErr(err)
	
	return block
}

func VerifyBlockWithoutChain(block *Block) error {
	errMSG := "Block verification failed."
	
	blockByte := block.Serialize()
	blockSize := uint(len(blockByte))
	
	if blockSize < uint(maxBlockHeaderPayload) || float32(blockSize) > float32(maxBlockSize) * stretch {
		return errors.New(errMSG)
	}
	
	err := VerifyTransactionsWithoutChain(block.Transactions)
	
	if err != nil {
		return err
	}
	
	err = VerifyBlockHeaderWithoutChain(block)
	
	if err != nil {
		return err
	}
	
	return nil
}