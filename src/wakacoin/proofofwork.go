package wakacoin

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math"
	"math/big"
	"time"
)

type ProofOfWork struct {
	blockHeader *BlockHeader
	target      *big.Int
}

func (pow *ProofOfWork) Run(bc *Blockchain, batch uint32, maxNonce uint64, sleep uint8) (uint64, error) {
	var nonce uint64 = 0
	var hashInt big.Int
	
	if maxNonce == 0 {
		maxNonce = math.MaxUint64
	} 
	
	if maxNonce < 1000 {
		maxNonce = 1000
	}
	
	if uint64(batch) > maxNonce {
		batch = uint32(maxNonce)
	}
	
	PrintMessage("Mining a new block...")
	
	for nonce < maxNonce {
		if miningPool {
			data := pow.prepareData(nonceFromMinersOfPool)
			
			hash := sha256.Sum256(data)
			hashInt.SetBytes(hash[:])
			
			if hashInt.Cmp(pow.target) == -1 {
				return nonceFromMinersOfPool, nil
			}
		}
		
		for i := 0; i < int(batch); i++ {
			data := pow.prepareData(nonce)

			hash := sha256.Sum256(data)
			hashInt.SetBytes(hash[:])

			if hashInt.Cmp(pow.target) == -1 {
				return nonce, nil
			} else {
				nonce++
			}
		}
		
		if sleep > 0 {
			time.Sleep(time.Duration(sleep) * time.Second)
		}
		
		if pow.blockHeader.PrevBlock != bc.tip {
			err := errors.New("The chain has been changed")
			return nonce, err
		}
	}
	
	err := errors.New("No solution")
	return nonce, err
}

func (pow *ProofOfWork) prepareData(nonce uint64) []byte {
	data := bytes.Join(
		[][]byte{
			Uint32ToByte(pow.blockHeader.Version),
			pow.blockHeader.PrevBlock[:],
			pow.blockHeader.MerkleRoot[:],
			Int64ToByte(pow.blockHeader.Timestamp),
			Uint8ToByte(pow.blockHeader.Difficulty),
			Uint64ToByte(nonce),
		},
		[]byte{},
	)

	return data
}

func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.blockHeader.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}

func NewProofOfWork(bh *BlockHeader) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, 256-uint(bh.Difficulty))

	pow := &ProofOfWork{bh, target}

	return pow
}
