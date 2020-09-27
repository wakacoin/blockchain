package wakacoin

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"time"
)

type BlockHeader struct {
	Version    uint32
	PrevBlock  [32]byte
	MerkleRoot [32]byte
	Timestamp  int64
	Difficulty uint8
	Nonce      uint64
}

func (bh *BlockHeader) HashBlockHeader() [32]byte {
	data := bytes.Join(
		[][]byte{
			Uint32ToByte(bh.Version),
			bh.PrevBlock[:],
			bh.MerkleRoot[:],
			Int64ToByte(bh.Timestamp),
			Uint8ToByte(bh.Difficulty),
			Uint64ToByte(bh.Nonce),
		},
		[]byte{},
	)	
	
	return sha256.Sum256(data)
}

func NewBlockHeader(prevBlock, merkleRoot [32]byte, difficulty uint8, bc *Blockchain, batch uint32, maxNonce uint64, sleep uint8) (*BlockHeader, error) {
	blockTime := time.Now().UTC().Unix()
	blockHeader := &BlockHeader{blockVersion, prevBlock, merkleRoot, blockTime, difficulty, 0}
	pow := NewProofOfWork(blockHeader)
	
	if miningPool {
		prepareData := bytes.Join(
			[][]byte{
				Uint32ToByte(pow.blockHeader.Version),
				pow.blockHeader.PrevBlock[:],
				pow.blockHeader.MerkleRoot[:],
				Int64ToByte(pow.blockHeader.Timestamp),
				Uint8ToByte(pow.blockHeader.Difficulty),
			},
			[]byte{},
		)
		
		SendMiningJob(WebServerLanAddress, prepareData, pow.blockHeader.Difficulty)
	}
	
	nonce, err := pow.Run(bc, batch, maxNonce, sleep)
	
	if err != nil {
		return blockHeader, err
	}
	
	blockHeader.Nonce = nonce
	
	return blockHeader, nil
}

func VerifyBlockHeaderWithoutChain(block *Block) (err error) {
	err = errors.New("BlockHeader verification failed.")
	
	if block.Header.Version != blockVersion {
		return err
	}
	
	txAmount := len(block.Transactions)
	var merkleRoot [32]byte
	
	if txAmount == 1 {
		merkleRoot = block.Transactions[0].ID
		
	} else if txAmount > 1 {
		merkleRoot = block.HashTransactions()
	}
	
	if block.Header.MerkleRoot != merkleRoot {
		return err
	}
	
	nodeTimeAddTenMinutes := time.Now().UTC().Add(+10 * time.Minute).Unix()
	
	if block.Header.Timestamp > nodeTimeAddTenMinutes {
		return err
	}
	
	pow := NewProofOfWork(block.Header)
	
	if isValid := pow.Validate(); !isValid {
		return err
	}
	
	return nil
}