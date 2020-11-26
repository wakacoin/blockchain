package wakacoin

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/boltdb/bolt"
)

type Blockchain struct {
	tip    [32]byte
	db     *bolt.DB
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{bc.tip, bc.db}
}

func (bc *Blockchain) VerifyBlock(block *Block, blockHashes *[][32]byte) error {
	for {
		if !isVerifying {
			break
		}
		
		PrintMessage("Waiting for block verification")
		time.Sleep(60 * time.Minute)
	}
	
	isVerifying = true
	PrintMessage("Block verification begins")
	
	defer func() {
		isVerifying = false
		PrintMessage("Block verification completed")
	}()
	
	blockHash := block.Header.HashBlockHeader()
	logExists := false
	isValid := false
	
	err := bc.db.View(func(tx *bolt.Tx) error {
		verifyB := tx.Bucket([]byte(verifyBucket))
		log := verifyB.Get(blockHash[:])
		
		if log != nil {
			logExists = true
			
			if bytes.Compare(log, []byte("S")) == 0 {
				isValid = true
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	if logExists {
		if isValid {
			return nil
			
		} else {
			return errors.New("Block verification failed.")
		}
	}
	
	err = bc.VerifyTransactions(block, blockHashes)
	
	if err != nil {
		bc.VerifyLog(blockHash, false)
		
		return err
	}
	
	err = bc.VerifyBlockHeader(block)
	
	if err != nil {
		bc.VerifyLog(blockHash, false)
		
		return err
	}
	
	bc.VerifyLog(blockHash, true)
	
	return nil
}

func (bc *Blockchain) VerifyLog(blockHash [32]byte, isValid bool) {
	err := bc.db.Update(func(tx *bolt.Tx) error {
		verifyB := tx.Bucket([]byte(verifyBucket))
		
		if isValid {
			err := verifyB.Put(blockHash[:], []byte("S"))
			CheckErr(err)
			
		} else {
			err := verifyB.Put(blockHash[:], []byte("F"))
			CheckErr(err)
		}
		
		return nil
	})
	CheckErr(err)
}

func (bc *Blockchain) VerifyTransactions(block *Block, blockHashes *[][32]byte) error {
	for index, _ := range block.Transactions {
		if index == 0 {
			err := bc.VerifyCoinbase(block)
			
			if err != nil {
				return err
			}
			
		} else {
			err := bc.VerifyTransaction(index, block, blockHashes)
			
			if err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (bc *Blockchain) VerifyCoinbase(block *Block) error {
	errMSG := "Coinbase verification failed."
	
	thisBlockHeight := block.GetHeight()
	
	if thisBlockHeight == 0 {
		var emptyArray [32]byte
		
		if block.Header.PrevBlock != emptyArray {
			return errors.New(errMSG)
		}
		
	} else {
		previousBlock, err := bc.GetBlock(block.Header.PrevBlock)
	
		if err != nil {
			return err
		}
		
		height := previousBlock.GetHeight()
		
		verification := height + 1
		
		if thisBlockHeight != verification {
			return errors.New(errMSG)
		}
	}
	
	return nil
}

func (bc *Blockchain) VerifyTransaction(index int, block *Block, blockHashes *[][32]byte) error {
	errMSG := "Transaction verification failed."
	
	if index < 1 {
		return errors.New(errMSG)
	}
	
	tnx := block.Transactions[index]
	thisBlockHeight := block.GetHeight()
	var totalInput, totalOutput uint
	
	for _, vin := range tnx.Vin {
		isFound := false
		
		for _, blockHash := range *blockHashes {
			if vin.Block == blockHash {
				isFound = true
			}
		}
		
		if isFound == false {
			return errors.New(errMSG)
		}
		
		var prevBlock *Block
		
		err := bc.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			blockData := b.Get(vin.Block[:])
			
			if blockData == nil {
				return errors.New("Block is not found")
			}
			
			prevBlock = DeserializeBlock(blockData)
			
			return nil
		})
		if err != nil {
			return err
		}
		
		prevBlockHeight := prevBlock.GetHeight()
		heightDifference := thisBlockHeight - prevBlockHeight
		
		if heightDifference <= uint32(spendableOutputConfirmations) {
			return errors.New(errMSG)
		}
		
		confirm := int(heightDifference) - int(spendableOutputConfirmations)
		bci := &BlockchainIterator{block.Header.PrevBlock, bc.db}
		
		for x := 0; x < confirm; x++ {
			b := bci.Next()
			
			for i, tx := range b.Transactions {
				if i != 0 {
					for _, v := range tx.Vin {
						if v.Block == vin.Block && v.Txid == vin.Txid && v.Index == vin.Index {
							return errors.New(errMSG)
						}
					}
				}
			}
		}
		
		isFound = false
		
		for _, tx := range prevBlock.Transactions {
			if tx.ID == vin.Txid {
				totalInput += tx.Vout[vin.Index].Value
				
				isFound = true
			}
		}
		
		if isFound == false {
			return errors.New("Transaction is not found")
		}
	}
	
	verifySign := tnx.VerifySign(bc)
	
	if verifySign != true {
		return errors.New(errMSG)
	}
	
	for _, vout := range tnx.Vout {
		totalOutput += vout.Value
	}
	
	totalInput--
	
	if totalInput != totalOutput {
		return errors.New(errMSG)
	}
	
	return nil
}

func (bc *Blockchain) VerifyBlockHeader(block *Block) error {
	errMSG := "BlockHeader verification failed."
	
	height := block.GetHeight()
	
	if height > 10 {
		timestamp := make([]int64, 11, 11)
		
		bci := &BlockchainIterator{block.Header.PrevBlock, bc.db}
		
		for i := 0; i < 11; i++ {
			b := bci.Next()
			timestamp[i] = b.Header.Timestamp
		}
		
		sort.Slice(timestamp, func(i, j int) bool {
			return timestamp[i] < timestamp[j]
		})
		
		if block.Header.Timestamp <= timestamp[5] {
			return errors.New(errMSG)
		}
	}
	
	validDifficulty := bc.GetValidDifficulty(block)
	
	if block.Header.Difficulty < validDifficulty {
		return errors.New(errMSG)
	}
	
	return nil
}

func (bc *Blockchain) SetDifficulty(height uint32, prevBlock [32]byte) uint8 {
	switch {
	case height < halving:
		return difficultyDefault_0
	
	case height < 32300:
		var timestamp [2]int64
		var currentDifficulty uint8
		x := int(averageBlockTimeOfBlocks)
		
		bci := &BlockchainIterator{prevBlock, bc.db}
		
		for i := 0; i <= x; i++ {
			block := bci.Next()
			
			if i == 0 {
				currentDifficulty = block.Header.Difficulty
				timestamp[0] = block.Header.Timestamp
				
				if timestamp[0] < time.Now().UTC().Add(-3 * time.Hour).Unix() {
					timeout := time.Now().UTC().Unix() - timestamp[0]
					var threshold int64 = 10800   //3 hours
					round := timeout / threshold
					weight := round * round
					
					if int64(currentDifficulty) > weight {
						if currentDifficulty - uint8(weight) > difficultyDefault_1 {
							difficulty := currentDifficulty - uint8(weight)
							
							return difficulty
							
						} else {
							return difficultyDefault_1
						}
						
					} else {
						return difficultyDefault_1
					}
				}
			}
			
			if i == x {
				timestamp[1] = block.Header.Timestamp
			}
		}
		
		interval := timestamp[0] - timestamp[1]
		average := interval / int64(averageBlockTimeOfBlocks)
		
		difficulty := currentDifficulty
		
		if average > int64(averageBlockTime + errorTolerance) {
			if difficulty > difficultyDefault_1 {
				difficulty--
			}
			
		} else if average < int64(averageBlockTime - errorTolerance) {
			difficulty++
		}
		
		return difficulty
	
	case height < 32600:
		var timestamp [2]int64
		var currentDifficulty uint8
		x := int(averageBlockTimeOfBlocks_2)
		
		bci := &BlockchainIterator{prevBlock, bc.db}
		
		for i := 0; i <= x; i++ {
			block := bci.Next()
			
			if i == 0 {
				currentDifficulty = block.Header.Difficulty
				timestamp[0] = block.Header.Timestamp
				
				if timestamp[0] < time.Now().UTC().Add(-3 * time.Hour).Unix() {
					timeout := time.Now().UTC().Unix() - timestamp[0]
					var threshold int64 = 10800   //3 hours
					round := timeout / threshold
					weight := round * round
					
					if int64(currentDifficulty) > weight {
						if currentDifficulty - uint8(weight) > difficultyDefault_1 {
							difficulty := currentDifficulty - uint8(weight)
							
							return difficulty
							
						} else {
							return difficultyDefault_1
						}
						
					} else {
						return difficultyDefault_1
					}
				}
			}
			
			if i == x {
				timestamp[1] = block.Header.Timestamp
			}
		}
		
		interval := timestamp[0] - timestamp[1]
		average := interval / int64(averageBlockTimeOfBlocks_2)
		
		difficulty := currentDifficulty
		
		if average > int64(averageBlockTime + errorTolerance_2) {
			if difficulty > difficultyDefault_1 {
				difficulty--
			}
			
		} else if average < int64(averageBlockTime - errorTolerance_2) {
			difficulty++
		}
		
		return difficulty
		
	default:
		var timestamp [2]int64
		var currentDifficulty uint8
		x := int(averageBlockTimeOfBlocks_2)
		
		bci := &BlockchainIterator{prevBlock, bc.db}
		
		for i := 0; i <= x; i++ {
			block := bci.Next()
			
			if i == 0 {
				currentDifficulty = block.Header.Difficulty
				timestamp[0] = block.Header.Timestamp
				
				if timestamp[0] < time.Now().UTC().Add(-3 * time.Hour).Unix() {
					timeout := time.Now().UTC().Unix() - timestamp[0]
					var threshold int64 = 10800   //3 hours
					round := timeout / threshold
					weight := round * round
					
					if int64(currentDifficulty) > weight {
						if currentDifficulty - uint8(weight) > difficultyDefault_1 {
							difficulty := currentDifficulty - uint8(weight)
							
							return difficulty
							
						} else {
							return difficultyDefault_1
						}
						
					} else {
						return difficultyDefault_1
					}
				}
			}
			
			if i == x {
				timestamp[1] = block.Header.Timestamp
			}
		}
		
		interval := timestamp[0] - timestamp[1]
		average := interval / int64(averageBlockTimeOfBlocks_2)
		
		difficulty := currentDifficulty
		
		if average > int64(averageBlockTime + errorTolerance_3) {
			if difficulty > difficultyDefault_1 {
				difficulty--
			}
			
		} else if average < int64(averageBlockTime - errorTolerance_3) {
			difficulty++
		}
		
		return difficulty
	}
}

func (bc *Blockchain) GetValidDifficulty(block *Block) uint8 {
	height := block.GetHeight()
	
	switch {
	case height < halving:
		return difficultyDefault_0
	
	case height < 32300:
		var timestamp [3]int64
		var referenceDifficulty uint8
		x := int(averageBlockTimeOfBlocks)
		timestamp[0] = block.Header.Timestamp
		
		bci := &BlockchainIterator{block.Header.PrevBlock, bc.db}
		
		for i := 0; i <= x; i++ {
			b := bci.Next()
			
			if i == 0 {
				timestamp[1] = b.Header.Timestamp
				referenceDifficulty = b.Header.Difficulty
			}
			
			if i == x {
				timestamp[2] = b.Header.Timestamp
			}
		}
		
		interval := timestamp[0] - timestamp[1]
		var threshold int64 = 10800   //3 hours
		
		if interval > threshold {
			round := interval / threshold
			weight := round * round
			
			if int64(referenceDifficulty) > weight {
				if referenceDifficulty - uint8(weight) > difficultyDefault_1 {
					difficulty := referenceDifficulty - uint8(weight)
					
					return difficulty
					
				} else {
					return difficultyDefault_1
				}
				
			} else {
				return difficultyDefault_1
			}
		}
		
		interval = timestamp[1] - timestamp[2]
		average := interval / int64(averageBlockTimeOfBlocks)
		
		difficulty := referenceDifficulty
		
		if average > int64(averageBlockTime + errorTolerance) {
			if difficulty > difficultyDefault_1 {
				difficulty--
			}
			
		} else if average < int64(averageBlockTime - errorTolerance) {
			difficulty++
		}
		
		return difficulty
	
	case height < 32600:
		var timestamp [3]int64
		var referenceDifficulty uint8
		x := int(averageBlockTimeOfBlocks_2)
		timestamp[0] = block.Header.Timestamp
		
		bci := &BlockchainIterator{block.Header.PrevBlock, bc.db}
		
		for i := 0; i <= x; i++ {
			b := bci.Next()
			
			if i == 0 {
				timestamp[1] = b.Header.Timestamp
				referenceDifficulty = b.Header.Difficulty
			}
			
			if i == x {
				timestamp[2] = b.Header.Timestamp
			}
		}
		
		interval := timestamp[0] - timestamp[1]
		var threshold int64 = 10800   //3 hours
		
		if interval > threshold {
			round := interval / threshold
			weight := round * round
			
			if int64(referenceDifficulty) > weight {
				if referenceDifficulty - uint8(weight) > difficultyDefault_1 {
					difficulty := referenceDifficulty - uint8(weight)
					
					return difficulty
					
				} else {
					return difficultyDefault_1
				}
				
			} else {
				return difficultyDefault_1
			}
		}
		
		interval = timestamp[1] - timestamp[2]
		average := interval / int64(averageBlockTimeOfBlocks_2)
		
		difficulty := referenceDifficulty
		
		if average > int64(averageBlockTime + errorTolerance_2) {
			if difficulty > difficultyDefault_1 {
				difficulty--
			}
			
		} else if average < int64(averageBlockTime - errorTolerance_2) {
			difficulty++
		}
		
		return difficulty
	
	default:
		var timestamp [3]int64
		var referenceDifficulty uint8
		x := int(averageBlockTimeOfBlocks_2)
		timestamp[0] = block.Header.Timestamp
		
		bci := &BlockchainIterator{block.Header.PrevBlock, bc.db}
		
		for i := 0; i <= x; i++ {
			b := bci.Next()
			
			if i == 0 {
				timestamp[1] = b.Header.Timestamp
				referenceDifficulty = b.Header.Difficulty
			}
			
			if i == x {
				timestamp[2] = b.Header.Timestamp
			}
		}
		
		interval := timestamp[0] - timestamp[1]
		var threshold int64 = 10800   //3 hours
		
		if interval > threshold {
			round := interval / threshold
			weight := round * round
			
			if int64(referenceDifficulty) > weight {
				if referenceDifficulty - uint8(weight) > difficultyDefault_1 {
					difficulty := referenceDifficulty - uint8(weight)
					
					return difficulty
					
				} else {
					return difficultyDefault_1
				}
				
			} else {
				return difficultyDefault_1
			}
		}
		
		interval = timestamp[1] - timestamp[2]
		average := interval / int64(averageBlockTimeOfBlocks_2)
		
		difficulty := referenceDifficulty
		
		if average > int64(averageBlockTime + errorTolerance_3) {
			if difficulty > difficultyDefault_1 {
				difficulty--
			}
			
		} else if average < int64(averageBlockTime - errorTolerance_3) {
			difficulty++
		}
		
		return difficulty
	}
}

func (bc *Blockchain) FindTXOutputPubKeyHash(blockHash, txid [32]byte, index int8) ([20]byte, error) {
	var block *Block
	var pubKeyHash [20]byte
	
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(blockHash[:])
		
		if encodedBlock == nil {
			return errors.New("Block is not found")
		}
		
		block = DeserializeBlock(encodedBlock)
		
		return nil
	})
	
	if err != nil {
		return pubKeyHash, err
	}
	
	for _, tx := range block.Transactions {
		if tx.ID == txid {
			pubKeyHash = tx.Vout[index].PubKeyHash
			
			return pubKeyHash, nil
		}
	}
	
	return pubKeyHash, errors.New("PubKeyHash is not found")
}

func (bc *Blockchain) GetBlock(blockHash [32]byte) (*Block, error) {
	var block *Block

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockData := b.Get(blockHash[:])

		if blockData == nil {
			return errors.New("The block is not found")
		}

		block = DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

func (bc *Blockchain) MineBlock(transactions []*Transaction, batch uint32, maxNonce uint64, sleep uint8) (*Block, error) {
	var block *Block
	
	b, err := bc.GetBlock(bc.tip)
	
	if err != nil {
		return block, err
	}
	
	bestHeight := b.GetHeight()
	
	newBlockHeight := bestHeight + 1
	cbTx := NewCoinbaseTX(MiningAddress, uint(len(transactions)), newBlockHeight)
	
	transactions = append([]*Transaction{cbTx}, transactions...)
	
	difficulty := bc.SetDifficulty(newBlockHeight, bc.tip)
	
	block, err = NewBlock(bc.tip, transactions, difficulty, bc, batch, maxNonce, sleep)
	
	if err != nil {
		return block, err
	}
	
	return block, nil
}

func (bc *Blockchain) VerifyBlockHashCollision(blockHash [32]byte) error {
	_, err := bc.GetBlock(blockHash)
	
	if err == nil {
		return errors.New("Hash Collision")
	}
	
	return nil
}

func (bc *Blockchain) AddLastBlock(block *Block) error {
	errMSG := "ERROR: The newly mined block is discarded because it is no longer in the main chain"
	
	if block.Header.PrevBlock != bc.tip {
		return errors.New(errMSG)
	}
	
	err := VerifyBlockWithoutChain(block)
	
	if err != nil {
		return err
	}
	
	err = bc.VerifyBlock(block, &blockHashesFromGenesisToTip)
	
	if err != nil {
		return err
	}
	
	if block.Header.PrevBlock != bc.tip {
		return errors.New(errMSG)
	}
	
	err = bc.db.Update(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		blocksB := tx.Bucket([]byte(blocksBucket))
		
		blockHash := block.Header.HashBlockHeader()
		
		err := blocksB.Put(blockHash[:], block.Serialize())
		CheckErr(err)
		
		err = parametersB.Put([]byte("lastBlock"), blockHash[:])
		CheckErr(err)
		
		bc.tip = blockHash
		
		return nil
	})
	CheckErr(err)
	
	if block.Header.PrevBlock == blockHashesFromGenesisToTip[(len(blockHashesFromGenesisToTip)-1)] {
		blockHashesFromGenesisToTip = append(blockHashesFromGenesisToTip, bc.tip)
	}
	
	length := int(block.GetHeight()) + 1
	
	if len(blockHashesFromGenesisToTip) != length {
		bc.RebuildBlockHashesFromGenesisToTip()
	}
	
	return nil
}

func (bc *Blockchain) Trim() {
	bci := bc.Iterator()
	
	for {
		blockHash := bci.currentHash
		block := bci.Next()
		
		blockHeight := block.GetHeight()
		
		if blockHeight <= protectHeight {
			break
			
		} else {
			for index, tx := range block.Transactions {
				if index != 0 {
					_ = tx.AddNewTxToBucket(bc.db)
				}
			}
			
			err := bc.db.Update(func(tx *bolt.Tx) error {
				parametersB := tx.Bucket([]byte(paramBucket))
				blocksB := tx.Bucket([]byte(blocksBucket))
				
				err := parametersB.Put([]byte("lastBlock"), block.Header.PrevBlock[:])
				CheckErr(err)
				
				bc.tip = block.Header.PrevBlock
				
				err = blocksB.Delete(blockHash[:])
				CheckErr(err)
				
				return nil
			})
			
			CheckErr(err)
		}
	}
}

func (bc *Blockchain) GetBlockHashes() *[][32]byte {
	PrintMessage("Generate a blockhashes slice begins")
	
	defer func() {
		PrintMessage("Generate a blockhashes slice completed")
	}()
	
	var blockHashes [][32]byte
	var emptyArray [32]byte
	bci := bc.Iterator()
	
	for {
		blockHash := bci.currentHash
		blockHashes = append(blockHashes, blockHash)
		
		block := bci.Next()
		
		if block.Header.PrevBlock == emptyArray {
			return &blockHashes
		}
	}
}

func (bc *Blockchain) UpdateMainChain(blockHash [32]byte, block *Block) {
	if block.Header.PrevBlock != bc.tip {
		return
	}
	
	err := bc.db.Update(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		err := parametersB.Put([]byte("lastBlock"), blockHash[:])
		CheckErr(err)
		
		bc.tip = blockHash
		
		return nil
	})
	CheckErr(err)
	
	if block.Header.PrevBlock == blockHashesFromGenesisToTip[(len(blockHashesFromGenesisToTip)-1)] {
		blockHashesFromGenesisToTip = append(blockHashesFromGenesisToTip, bc.tip)
	}
	
	length := int(block.GetHeight()) + 1
	
	if len(blockHashesFromGenesisToTip) != length {
		bc.RebuildBlockHashesFromGenesisToTip()
	}
	
	UTXOSet := UTXOSet{bc}
	err = UTXOSet.UpdateLastBlock()
	
	switch {
	case err == nil:
		_ = UTXOSet.UpdateLastBlock()
		PrintMessage("Update UTXO completed")
		
	case err.Error() == utxoHaveToBeReindexed:
		_ = UTXOSet.Reindex()
		_ = UTXOSet.Reindex()
	}
}

func (bc *Blockchain) ChangeMainChain(bestHeight uint32, lastHash [32]byte) {
	topBlock, err := bc.GetBlock(bc.tip)
	CheckErr(err)
	
	bcBestHeight := topBlock.GetHeight()
	
	if bcBestHeight >= bestHeight {
		return
	}
	
	bci_oldChain := bc.Iterator()
	
	err = bc.db.Update(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		err := parametersB.Put([]byte("lastBlock"), lastHash[:])
		CheckErr(err)
		
		bc.tip = lastHash
		
		return nil
	})
	CheckErr(err)
	
	var blockHashes [][32]byte
	isFound := false
	length := 0
	var emptyArray [32]byte
	bci := bc.Iterator()
	
	for {
		blockHash := bci.currentHash
		block := bci.Next()
		
		for i, v := range blockHashesFromGenesisToTip {
			if blockHash == v {
				isFound = true
				length = i + 1
				ReverseHashes(blockHashes)
			}
			
			if isFound {
				break
			}
		}
		
		if !isFound {
			blockHashes = append(blockHashes, blockHash)
			
		} else {
			tmp := make([][32]byte, length)
			copy(tmp, blockHashesFromGenesisToTip[:length])
			
			tmp = append(tmp, blockHashes...)
			copy(blockHashesFromGenesisToTip, tmp)
			
			blockHashesFromGenesisToTip = append(blockHashesFromGenesisToTip, tmp[len(blockHashesFromGenesisToTip):]...)
			
			break
		} 
		
		if block.Header.PrevBlock == emptyArray {
			bc.RebuildBlockHashesFromGenesisToTip()
			
			break
		}
	}
	
	UTXOSet := UTXOSet{bc}
	
	for i := 1; i < 9; i++ {
		err := UTXOSet.Reindex()
		
		if err == nil {
			break
			
		} else {
			PrintMessage("UTXO is waiting to be reindexed.")
		}
		
		sleep := i * 2
		time.Sleep(time.Duration(sleep) * time.Minute)
	}
	
	counter := 0
	
	for {
		block := bci_oldChain.Next()
		
		if counter == 0 {
			blockHash := block.Header.HashBlockHeader()
			
			for _, v := range blockHashesFromGenesisToTip {
				if blockHash == v {
					return
				}
			}
		}
		
		counter++
		
		for index, tx := range block.Transactions {
			if index != 0 {
				isSpendable, err := UTXOSet.isSpendableTX(tx)
				
				isOK := false
	
				if isSpendable {
					isOK = true
					
				} else {
					if err.Error() == confirmationsError {
						isOK = true
					}
				}
				
				if isOK {
					_ = tx.AddNewTxToBucket(bc.db)
				}
			}
		}
		
		if block.Header.PrevBlock == emptyArray {
			return
		}
		
		for _, v := range blockHashesFromGenesisToTip {
			if block.Header.PrevBlock == v {
				return
			}
		}
	}
}

func (bc *Blockchain) RebuildBlockHashesFromGenesisToTip() {
	if isRebuilding {
		return
	}
	
	isRebuilding = true
	PrintMessage("Rebuild blockHashesFromGenesisToTip begins")
	
	defer func() {
		isRebuilding = false
		PrintMessage("Rebuild blockHashesFromGenesisToTip completed")
	}()
	
	tmp := *bc.GetBlockHashes()
	ReverseHashes(tmp)
	tmpLeng := len(tmp)
	
	topBlock, err := bc.GetBlock(bc.tip)
	CheckErr(err)
	
	bestHeight := topBlock.GetHeight()
	
	if int(bestHeight + 1) == tmpLeng {
		copy(blockHashesFromGenesisToTip, tmp)
		
		length := len(blockHashesFromGenesisToTip)
		
		switch {
		case length < tmpLeng:
			blockHashesFromGenesisToTip = append(blockHashesFromGenesisToTip, tmp[length:]...)
			
		case length > tmpLeng:
			blockHashesFromGenesisToTip = blockHashesFromGenesisToTip[:tmpLeng] 
		}
	}
}

func (bc *Blockchain) VerifyProtect() error {
	errMSG := "Protect verification failed."
	
	var protectG, protectT [32]byte
	var protectGExists, protectTExists bool
	
	slice, err := hex.DecodeString(protectFromGenesis)
	CheckErr(err)
	
	copy(protectG[:], slice)
	
	slice, err = hex.DecodeString(protectTo)
	CheckErr(err)
	
	copy(protectT[:], slice)
	
	bci := bc.Iterator()
	var emptyArray [32]byte
	
	for {
		blockHash := bci.currentHash
		block := bci.Next()
		
		if blockHash == protectT {
			if block.GetHeight() == protectHeight {
				protectTExists = true
			}
		}
		
		if block.Header.PrevBlock == emptyArray {
			if blockHash == protectG {
				protectGExists = true
			}
			
			if protectGExists && protectTExists {
				return nil
				
			} else {
				return errors.New(errMSG)
			}
		}
	}
}

func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func NewBlockchain(nodeID string) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	
	if dbExists(dbFile) == false {
		errMSG := "ERROR: No existing database found. Please download the latest version of the database from https://each1.net/public/wakacoin/"
		
		fmt.Println("\n", errMSG)
		os.Exit(1)
	}
	
	var lastBlock []byte
	
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	CheckErr(err)
	
	err = db.Update(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		lastBlock = parametersB.Get([]byte("lastBlock"))
		
		if lastBlock == nil {
			errMSG := "ERROR: No existing last block hash found. Please download the latest version of the database from https://each1.net/public/wakacoin/"
			
			fmt.Println("\n", errMSG)
			os.Exit(1)
		}
		
		return nil
	})
	CheckErr(err)
	
	var tip [32]byte
	copy(tip[:], lastBlock)
	
	bc := Blockchain{tip, db}
	
	return &bc
}
