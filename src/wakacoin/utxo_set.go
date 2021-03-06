package wakacoin

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	
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
	Value  uint
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

func (u UTXOSet) GetAvailableUTXO() (utxoBucket []byte) {
	err := u.Blockchain.db.View(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		
		utxoBucket = parametersB.Get([]byte("availableUTXO"))
		
		if utxoBucket == nil {
			return errors.New("Error: The available UTXO is not found")
		}
		
		return nil
	})
	CheckErr(err)
	
	return
}

func (u UTXOSet) GetUnAvailableUTXO() (utxoBucket []byte) {
	availableUTXO := u.GetAvailableUTXO()
	
	if bytes.Compare(availableUTXO, []byte("utxoBucket_A")) == 0 {
		utxoBucket = []byte("utxoBucket_B")
		
	} else {
		utxoBucket = []byte("utxoBucket_A")
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
	if bytes.Compare(utxoBucket, []byte("utxoBucket_A")) == 0 {
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
	
	utxoBucket := u.GetUnAvailableUTXO()
	
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
	u.Update(utxoBucket, topBlock)
	u.SetAvailable(utxoBucket, top)
	
	return nil
}

func (u UTXOSet) Reindex() error {
	utxoBucket := u.GetUnAvailableUTXO()
	
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
		err := u.Rebuild(utxoBucket)
		CheckErr(err)
		
	} else {
		for i := utxoLastBlockID + 1; i < length; i++ {
			block, err := u.Blockchain.GetBlock(blockHashes[i])
			CheckErr(err)
			
			u.Update(utxoBucket, block)
		}
	}
	
	x := length - 1
	u.SetAvailable(utxoBucket, blockHashes[x])
	
	return nil
}

func (u UTXOSet) Rebuild(utxoBucket []byte) error {
	err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(utxoBucket)
		
		if err != nil && err != bolt.ErrBucketNotFound {
			CheckErr(err)
		}
		
		_, err = tx.CreateBucket(utxoBucket)
		CheckErr(err)
		
		return nil
	})
	CheckErr(err)
	
	blockHashes := blockHashesFromGenesisToTip
	length := len(blockHashes)
	
	for i := 0; i < length; i++ {
		block, err := u.Blockchain.GetBlock(blockHashes[i])
		CheckErr(err)
		
		u.Update(utxoBucket, block)
	}
	
	return nil
}

func (u UTXOSet) Update(utxoBucket []byte, block *Block) {
	blockHeight := block.GetHeight()
	blockHash := block.Header.HashBlockHeader()
	blockHashString := hex.EncodeToString(blockHash[:])
	
	err := u.Blockchain.db.Update(func(tx *bolt.Tx) error {
		utxoB := tx.Bucket(utxoBucket)
		txsB := tx.Bucket([]byte(txsBucket))
		
		for index, tx := range block.Transactions {
			utxoID := hex.EncodeToString(tx.ID[:]) + "@" + blockHashString
			
			if index != 0 {
				for _, vin := range tx.Vin {
					inTxoID := hex.EncodeToString(vin.Txid[:]) + "@" + hex.EncodeToString(vin.Block[:])
					
					updatedOuts := UTXOutputs{}
					outsBytes := utxoB.Get([]byte(inTxoID))
					outs := DeserializeUTXOutputs(outsBytes)

					for _, out := range outs.Outputs {
						if out.Index != vin.Index {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := utxoB.Delete([]byte(inTxoID))
						CheckErr(err)
						
					} else {
						err := utxoB.Put([]byte(inTxoID), updatedOuts.Serialize())
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
				uTXOutput := &UTXOutput{int8(outIdx), out.Value, out.PubKeyHash, blockHeight}
				newOutputs.Outputs = append(newOutputs.Outputs, uTXOutput)
			}
			
			err := utxoB.Put([]byte(utxoID), newOutputs.Serialize())
			CheckErr(err)
		}
		
		return nil
	})
	CheckErr(err)
}

func (u UTXOSet) Balance(pubKeyHash [20]byte) (balance, balanceSpendable uint) {
	utxoBucket := u.GetAvailableUTXO()
	
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
	
	return
}

func (u UTXOSet) FindSpendableOutputs(usableWallets *Wallets, amountAndFee uint) (uint, *ValidOutputs, error) {
	if amountAndFee < 2 {
		fmt.Println(time.Now().UTC())
		panic("ERROR: Amount is not valid.")
	}
	
	var accumulated, balanceSpendable uint
	var validOutputs ValidOutputs
	var errMSG string
	var spendedOutputs SpendedOutputs
	var spendableOutputs SpendableOutputs
	sliceOutputs := []uint{}
	
	topBlock, err := u.Blockchain.GetBlock(u.Blockchain.tip)
	CheckErr(err)
	
	bestHeight := topBlock.GetHeight()
	
	utxoBucket := u.GetAvailableUTXO()
	
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
			utxoID := string(k)
			outs := DeserializeUTXOutputs(v)
			
			ID := strings.Split(utxoID, "@")
			
			var txID [32]byte
			
			decoded, err := hex.DecodeString(ID[0])
			CheckErr(err)
			
			copy(txID[:], decoded)
			
			var block [32]byte
			
			decoded, err = hex.DecodeString(ID[1])
			CheckErr(err)
			
			copy(block[:], decoded)
			
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
								if out.Value >= amountAndFee {
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
		
		sort.Slice(sliceOutputs, func(i, j int) bool {
			return sliceOutputs[i] < sliceOutputs[j]
		})
		
		var outsValue, outValue, counter uint
		
		for _, v := range sliceOutputs {
			if outsValue < amountAndFee {
				outsValue += v
				outValue = v
				
			} else {
				break
			}
			
			counter++
			
			if counter >= uint(maxVins) {
				break
			}
		}
		
		if outsValue >= amountAndFee {
			for _, out := range spendableOutputs.Outputs {
				if accumulated < amountAndFee {
					if out.Value < outValue {
						accumulated += out.Value
						validOutput := &ValidOutput{out.Block, out.Txid, out.Index, out.PubKey}
						validOutputs.Outputs = append(validOutputs.Outputs, validOutput)
					}
					
				} else {
					return nil
				}
			}
			
			for _, out := range spendableOutputs.Outputs {
				if accumulated < amountAndFee {
					if out.Value == outValue {
						accumulated += out.Value
						validOutput := &ValidOutput{out.Block, out.Txid, out.Index, out.PubKey}
						validOutputs.Outputs = append(validOutputs.Outputs, validOutput)
					}
					
				} else {
					return nil
				}
			}
			
		} else {
			outsValue = 0
			outValue = 0
			
			sliceOutputsLength := len(sliceOutputs)
			
			for i := (sliceOutputsLength - 1); i >= (sliceOutputsLength - int(maxVins)); i-- {
				if i < 0 {
					break
				}
				
				if outsValue < amountAndFee {
					outsValue += sliceOutputs[i]
					outValue = sliceOutputs[i]
					
				} else {
					break
				}
			}
			
		}
		
		if outsValue < amountAndFee {
			amount := int(amountAndFee - 1)
			
			errMSG = "ERROR: The maximum number of inputs is " + strconv.Itoa(int(maxVins)) + " per transaction. It’s unable to collect " + strconv.Itoa(amount) + " wakacoins from your wallet within " + strconv.Itoa(int(maxVins)) + " inputs. Please transfer " + strconv.Itoa(amount) + " wakacoins in two or more batches."
			
			if !miningPool {
				fmt.Println("\n", errMSG)
				os.Exit(1)
			}
			
			return nil
			
		} else {
			for _, out := range spendableOutputs.Outputs {
				if accumulated < amountAndFee {
					if out.Value > outValue {
						accumulated += out.Value
						validOutput := &ValidOutput{out.Block, out.Txid, out.Index, out.PubKey}
						validOutputs.Outputs = append(validOutputs.Outputs, validOutput)
					}
					
				} else {
					return nil
				}
			}
			
			for _, out := range spendableOutputs.Outputs {
				if accumulated < amountAndFee {
					if out.Value == outValue {
						accumulated += out.Value
						validOutput := &ValidOutput{out.Block, out.Txid, out.Index, out.PubKey}
						validOutputs.Outputs = append(validOutputs.Outputs, validOutput)
					}
					
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
	
	utxoBucket := u.GetAvailableUTXO()
	
	inUseID := AddInUse(utxoBucket)
	
	defer func() {
		inUse[inUseID] = ""
	}()
	
	err = u.Blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxoBucket)
		
		for _, vin := range tnx.Vin {
			inTxoID := hex.EncodeToString(vin.Txid[:]) + "@" + hex.EncodeToString(vin.Block[:])
			outsBytes := b.Get([]byte(inTxoID))
			
			if outsBytes == nil {
				return errors.New("Error: Unspent transaction output is not found")
			}
			
			outs := DeserializeUTXOutputs(outsBytes)
			outputExists := false
			confirmationsEnough := false
			
			for _, out := range outs.Outputs {
				if out.Index == vin.Index {
					outputExists = true
					
					if (bestHeight - out.Height) > uint32(spendableOutputConfirmations) {
						confirmationsEnough = true
						
						break							
					}
				}
			}
			
			if outputExists != true {
				return errors.New("Error: Unspent transaction output is not found")
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