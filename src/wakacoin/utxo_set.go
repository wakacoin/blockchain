package wakacoin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	
	"github.com/boltdb/bolt"
)

type UTXOSet struct {
	Blockchain *Blockchain
}

type ValidOutput struct {
	Block  [32]byte
	Txid   [32]byte
	Index  int8
	PubKey []byte
}

type ValidOutputs struct {
	Outputs []*ValidOutput
}

type SpendedOutput struct {
	Block [32]byte
	Txid  [32]byte
	Index int8
}

type SpendedOutputs struct {
	Outputs []*SpendedOutput
}

type SpendableOutput struct {
	Block  [32]byte
	Txid   [32]byte
	Index  int8
	Value  uint32
	PubKey []byte
}

type SpendableOutputs struct {
	Outputs []*SpendableOutput
}

func (u UTXOSet) ResetUTXOLastBlock(utxoBucket []byte) {
	err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		
		err := parametersB.Delete(utxoBucket)
		CheckErr(err)
		
		return nil
	})
	CheckErr(err)
	
	PrintMessage("Reindex UTXO begins")
}

func (u UTXOSet) SetAvailable(utxoBucket []byte, utxoLastBlockHash [32]byte) {
	err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		
		err := parametersB.Put(utxoBucket, utxoLastBlockHash[:])
		CheckErr(err)
		
		err = parametersB.Put([]byte("availableUTXO"), utxoBucket)
		CheckErr(err)
		
		return nil
	})
	CheckErr(err)
	
	PrintMessage("Reindex UTXO completed")
}

func (u UTXOSet) GetAvailableUTXO() (utxoBucket, contractBucket []byte) {
	err := u.Blockchain.db.View(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		
		utxoBucket = parametersB.Get([]byte("availableUTXO"))
		
		if utxoBucket == nil {
			return errors.New("Error: The available UTXO is not found")
		}

		if bytes.Compare(utxoBucket, []byte(utxoBucket_A)) == 0 {
			contractBucket = []byte(contractBucket_A)
		} else {
			contractBucket = []byte(contractBucket_B)
		} 
		
		return nil
	})
	CheckErr(err)
	
	return
}

func (u UTXOSet) GetUnAvailableUTXO() (utxoBucket, contractBucket []byte) {
	availableUTXO, _ := u.GetAvailableUTXO()
	
	if bytes.Compare(availableUTXO, []byte(utxoBucket_A)) == 0 {
		utxoBucket = []byte(utxoBucket_B)
		contractBucket = []byte(contractBucket_B)
		
	} else {
		utxoBucket = []byte(utxoBucket_A)
		contractBucket = []byte(contractBucket_A)
	} 
	
	return
}

func (u UTXOSet) GetUTXOLastBlock(utxoBucket []byte) (utxoLastBlockHash [32]byte) {
	err := u.Blockchain.db.View(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		
		hash := parametersB.Get(utxoBucket)
		
		if hash != nil {
			copy(utxoLastBlockHash[:], hash)
		}
		
		return nil
	})
	CheckErr(err)
	
	return
}

func AddInUse(utxoBucket []byte) (inUseID int) {
	utxoBucketID := GetUTXOBucketID(utxoBucket)
	
	var findEmpty bool
	
	for i, v := range inUse {
		if v == "" {
			findEmpty = true
			inUseID = i
			break
		}
	}
	
	if findEmpty {
		inUse[inUseID] = utxoBucketID
		
	} else {
		inUse = append(inUse, utxoBucketID)
		inUseID = len(inUse) - 1
	}
	
	return
}

func IsInUse(utxoBucket []byte) (isInUse bool) {
	utxoBucketID := GetUTXOBucketID(utxoBucket)
	
	for _, v := range inUse {
		if v == utxoBucketID {
			isInUse = true
			
			return
		}
	}
	
	return
}

func GetUTXOBucketID(utxoBucket []byte) (utxoBucketID string) {
	if bytes.Compare(utxoBucket, []byte(utxoBucket_A)) == 0 {
		utxoBucketID = "A"
		
	} else {
		utxoBucketID = "B"
	}
	
	return
}

func (u UTXOSet) UpdateLastBlock() error {
	if isUpdating {
		return errors.New(utxoCanNotBeUpdated)
	}
	
	utxoBucket, contractBucket := u.GetUnAvailableUTXO()
	
	isInUse := IsInUse(utxoBucket)
	
	if isInUse {
		return errors.New(utxoCanNotBeUpdated)
	}
	
	utxoLastBlock := u.GetUTXOLastBlock(utxoBucket)
	
	if utxoLastBlock == u.Blockchain.tip {
		return nil
	}
	
	top := u.Blockchain.tip
	topBlock, err := u.Blockchain.GetBlock(top)
	CheckErr(err)
	
	if topBlock.Header.PrevBlock != utxoLastBlock {
		return errors.New(utxoHaveToBeReindexed)
	}
	
	isUpdating = true
	
	defer func() {
		isUpdating = false
	}()
	
	u.ResetUTXOLastBlock(utxoBucket)
	u.Update(utxoBucket, contractBucket, topBlock)
	u.SetAvailable(utxoBucket, top)
	
	return nil
}

func (u UTXOSet) Reindex() error {
	utxoBucket, contractBucket := u.GetUnAvailableUTXO()
	
	if isUpdating {
		return errors.New(utxoCanNotBeUpdated)
	}
	
	isInUse := IsInUse(utxoBucket)

	if isInUse {
		return errors.New(utxoCanNotBeUpdated)
	}
	
	utxoLastBlock := u.GetUTXOLastBlock(utxoBucket)
	
	if utxoLastBlock == u.Blockchain.tip {
		return nil
	}
	
	isUpdating = true
	
	defer func() {
		isUpdating = false
	}()
	
	var emptyArray [32]byte
	var utxoLastBlockID int
	haveToRebuild := true
	blockHashes := blockHashesFromGenesisToTip
	
	if utxoLastBlock != emptyArray {
		for i, blockHash := range blockHashes {
			if utxoLastBlock == blockHash {
				haveToRebuild = false
				utxoLastBlockID = i
			}
		}
	}
	
	u.ResetUTXOLastBlock(utxoBucket)
	
	length := len(blockHashes)
	
	if haveToRebuild {
		err := u.Rebuild(utxoBucket, contractBucket)
		CheckErr(err)
		
	} else {
		for i := utxoLastBlockID + 1; i < length; i++ {
			block, err := u.Blockchain.GetBlock(blockHashes[i])
			CheckErr(err)
			
			u.Update(utxoBucket, contractBucket, block)
		}
	}
	
	x := length - 1
	u.SetAvailable(utxoBucket, blockHashes[x])
	
	return nil
}

func (u UTXOSet) Rebuild(utxoBucket, contractBucket []byte) error {
	err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(utxoBucket)
		
		if err != nil && err != bolt.ErrBucketNotFound {
			CheckErr(err)
		}

		err = tx.DeleteBucket(contractBucket)
		
		if err != nil && err != bolt.ErrBucketNotFound {
			CheckErr(err)
		}
		
		_, err = tx.CreateBucket(utxoBucket)
		CheckErr(err)
		
		_, err = tx.CreateBucket(contractBucket)
		CheckErr(err)
		
		return nil
	})
	CheckErr(err)
	
	blockHashes := blockHashesFromGenesisToTip
	length := len(blockHashes)
	
	for i := 0; i < length; i++ {
		block, err := u.Blockchain.GetBlock(blockHashes[i])
		CheckErr(err)
		
		u.Update(utxoBucket, contractBucket, block)
	}
	
	return nil
}

func (u UTXOSet) Update(utxoBucket, contractBucket []byte, block *Block) {
	blockHeight := block.GetHeight()
	blockHash := block.Header.HashBlockHeader()
	
	err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		utxoB := tx.Bucket(utxoBucket)
		contractB := tx.Bucket(contractBucket)
		txsB := tx.Bucket([]byte(txsBucket))
		
		for index, tx := range block.Transactions {
			utxoID :=  append(blockHash[:], tx.ID[:]...)
			
			if index != 0 {
				for _, vin := range tx.Vin {
					inTxoID := append(vin.Block[:], vin.Txid[:]...)
					
					updatedOuts := UTXOutputs{}
					outsBytes := utxoB.Get(inTxoID)
					outs := DeserializeUTXOutputs(outsBytes)

					for _, out := range outs.Outputs {
						if out.Index != vin.Index {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
							break
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := utxoB.Delete(inTxoID)
						CheckErr(err)
						
					} else {
						err := utxoB.Put(inTxoID, updatedOuts.Serialize())
						CheckErr(err)
					}
					
					c := txsB.Cursor()
					
					for k, v := c.First(); k != nil; k, v = c.Next() {
						tranx := DeserializeTransaction(v)
						
						for _, in := range tranx.Vin {
							if in.Block == vin.Block && in.Txid == vin.Txid && in.Index == vin.Index {
								err := txsB.Delete(k)
								CheckErr(err)
								
								break
							}
						}
					}
				}
			}
			
			newOutputs := UTXOutputs{}
			
			for outIdx, out := range tx.Vout {
				if out.Value > 0 {
					uTXOutput := &UTXOutput{int8(outIdx), out.Value, out.PubKeyHash, blockHeight}
					newOutputs.Outputs = append(newOutputs.Outputs, uTXOutput)
				
				} else {
					if index != 0 && outIdx == 0 {
						a1 := HashPubKey(tx.Vin[0].PubKey)
						a2 := out.PubKeyHash
						s1 := a1[:]
						s2 := a2[:]
						s1 = append(s1, s2...)

						err := contractB.Put(s1, nil)
						CheckErr(err)
					}
				}
			}
			
			if len(newOutputs.Outputs) > 0 {
				err := utxoB.Put(utxoID, newOutputs.Serialize())
				CheckErr(err)
			}
		}
		
		return nil
	})
	CheckErr(err)
}

func (u UTXOSet) Balance(pubKeyHash [20]byte, printDetail bool) (balance, balanceSpendable uint32) {
	var values []uint32

	utxoBucket, _ := u.GetAvailableUTXO()
	
	topBlock, err := u.Blockchain.GetBlock(u.Blockchain.tip)
	CheckErr(err)
	
	bestHeight := topBlock.GetHeight()
	
	err = u.Blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxoBucket)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeUTXOutputs(v)

			for _, out := range outs.Outputs {
				if out.PubKeyHash == pubKeyHash {
					if printDetail {
						values = append(values, out.Value)
					}

					balance += out.Value
					
					if bestHeight > out.Height {
						if (bestHeight - out.Height) > uint32(spendableOutputConfirmations) {
							balanceSpendable += out.Value
						}
					}
				}
			}
		}

		return nil
	})
	CheckErr(err)
	
	if printDetail {
		sort.Slice(values, func(i, j int) bool {
			return values[i] < values[j]
		})

		for _, v := range values {
			fmt.Println(v)
		}
	}

	return
}

func (u UTXOSet) FindSpendableOutputs(usableWallets *Wallets, amountAndFee uint32) (uint32, *ValidOutputs, error) {
	var accumulated uint32
	var validOutputs ValidOutputs

	if amountAndFee < 1 {
		return accumulated, &validOutputs, errors.New("ERROR: The amountAndFee is not valid.")
	}
	
	var balanceSpendable uint32
	var errMSG string
	var spendedOutputs SpendedOutputs
	var spendableOutputs SpendableOutputs
	sliceOutputs := []uint32{}
	
	topBlock, err := u.Blockchain.GetBlock(u.Blockchain.tip)
	
	if err != nil {
		err = errors.New("ERROR: The topBlock is not found when FindSpendableOutputs.")
		CheckErr(err)
	}
	
	bestHeight := topBlock.GetHeight()
	
	utxoBucket, _ := u.GetAvailableUTXO()
	
	inUseID := AddInUse(utxoBucket)
	
	defer func() {
		inUse[inUseID] = ""
	}()
	
	err = u.Blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(txsBucket))
		c := b.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			tnx := DeserializeTransaction(v)
			
			for _, vin := range tnx.Vin {
				for _, wallet := range usableWallets.Wallets {
					if bytes.Compare(vin.PubKey, wallet.PublicKey) == 0 {
						spendedOutput := &SpendedOutput{vin.Block, vin.Txid, vin.Index}
						spendedOutputs.Outputs = append(spendedOutputs.Outputs, spendedOutput)
					}
				}
			}
		}
		
		b = tx.Bucket(utxoBucket)
		c = b.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeUTXOutputs(v)
			
			var block [32]byte
			copy(block[:], k[:32])

			var txID [32]byte
			copy(txID[:], k[32:])
			
			for _, out := range outs.Outputs {
				if (bestHeight - out.Height) > uint32(spendableOutputConfirmations) {
					for _, wallet := range usableWallets.Wallets {
						pubKeyHash := HashPubKey(wallet.PublicKey)
						
						if out.PubKeyHash == pubKeyHash {
							isSpended := false
							
							for _, spended := range spendedOutputs.Outputs {
								if block == spended.Block && txID == spended.Txid && out.Index == spended.Index {
									isSpended = true
								}
							}
							
							if !isSpended {
								if out.Value == amountAndFee {
									accumulated = out.Value
									validOutput := &ValidOutput{block, txID, out.Index, wallet.PublicKey}
									validOutputs.Outputs = append(validOutputs.Outputs, validOutput)
									
									return nil
								}

								balanceSpendable += out.Value
								
								spendableOutput := &SpendableOutput{block, txID, out.Index, out.Value, wallet.PublicKey}
								spendableOutputs.Outputs = append(spendableOutputs.Outputs, spendableOutput)
								
								sliceOutputs = append(sliceOutputs, out.Value)
							}
						}
					}
				}
			}
		}
		
		if balanceSpendable < amountAndFee {
			errMSG = "ERROR: Your spendable balance is insufficient."
			
			if !miningPool {
				fmt.Println("\n", errMSG)
				os.Exit(1)
			}
			
			return nil
		}

		var numbers uint16 = 30

		if numbers > maxVins - 1 {
			numbers = maxVins - 1
		}

		accumulated = 0

		for i, out := range spendableOutputs.Outputs {
			if i < int(numbers) {
				accumulated += out.Value

			} else {
				break
			}
		}

		if accumulated >= amountAndFee {
			accumulated = 0

			for _, out := range spendableOutputs.Outputs {
				if accumulated < amountAndFee {
					accumulated += out.Value
					validOutput := &ValidOutput{out.Block, out.Txid, out.Index, out.PubKey}
					validOutputs.Outputs = append(validOutputs.Outputs, validOutput)

				} else {
					return nil
				}
			}

		} else {
			sort.Slice(spendableOutputs.Outputs, func(i, j int) bool {
				return spendableOutputs.Outputs[i].Value > spendableOutputs.Outputs[j].Value
			})

			accumulated = 0

			for i, out := range spendableOutputs.Outputs {
				if i >= int(maxVins) {
					amount := int(amountAndFee - 1)
			
					errMSG = "ERROR: The maximum number of inputs is " + strconv.Itoa(int(maxVins)) + " per transaction. It’s unable to collect " + strconv.Itoa(amount) + " wakacoins from your wallet within " + strconv.Itoa(int(maxVins)) + " inputs. Please transfer " + strconv.Itoa(amount) + " wakacoins in several batches."
					
					if !miningPool {
						fmt.Println("\n", errMSG)
						os.Exit(1)
					}
					
					return nil
				}

				if accumulated < amountAndFee {
					accumulated += out.Value
					validOutput := &ValidOutput{out.Block, out.Txid, out.Index, out.PubKey}
					validOutputs.Outputs = append(validOutputs.Outputs, validOutput)

				} else {
					return nil
				}
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	if len(errMSG) == 0 {
		return accumulated, &validOutputs, nil
		
	} else {
		return accumulated, &validOutputs, errors.New(errMSG)
	}
}

func (u UTXOSet) isSpendableTX(tnx *Transaction) (bool, error) {
	if tnx.IsCoinbase() {
		return false, errors.New("Error: is coinbase")
	}
	
	topBlock, err := u.Blockchain.GetBlock(u.Blockchain.tip)
	CheckErr(err)
	
	bestHeight := topBlock.GetHeight()
	
	utxoBucket, _ := u.GetAvailableUTXO()
	
	inUseID := AddInUse(utxoBucket)
	
	defer func() {
		inUse[inUseID] = ""
	}()
	
	err = u.Blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxoBucket)
		
		for _, vin := range tnx.Vin {
			inTxoID := append(vin.Block[:], vin.Txid[:]...)
			outsBytes := b.Get(inTxoID)
			
			if outsBytes == nil {
				return errors.New("outsBytes nil. Unspent transaction output is not found. The transaction may have already been written into the blockchain. The transaction will be discarded.")
			}
			
			outs := DeserializeUTXOutputs(outsBytes)
			outputExists := false
			confirmationsEnough := false
			
			for _, out := range outs.Outputs {
				if out.Index == vin.Index {
					outputExists = true
					
					if (bestHeight - out.Height) > uint32(spendableOutputConfirmations) {
						confirmationsEnough = true							
					}

					break
				}
			}
			
			if outputExists != true {
				return errors.New("outputExists false. Unspent transaction output is not found. The transaction may have already been written into the blockchain. The transaction will be discarded.")
			}
			
			if confirmationsEnough != true {
				return errors.New(confirmationsError)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return false, err
	}
	
	verifySign := tnx.VerifySign(u.Blockchain)
	
	if verifySign {
		return true, nil
		
	} else {
		return false, errors.New("Error: Signature verification failed")
	}
}