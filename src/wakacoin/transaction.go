package wakacoin

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	
	"github.com/boltdb/bolt"
)

type Transaction struct {
	ID      [32]byte
	Version uint32
	Vin     []TXInput
	Vout    []TXOutput
}

func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("- Transaction    %x", tx.ID))
	lines = append(lines, fmt.Sprintf("  Version        %x", tx.Version))
	
	counter := len(tx.Vin)
	
	i := 0
	input := tx.Vin[i]
	lines = append(lines, fmt.Sprintf("   Input %d", i))
	lines = append(lines, fmt.Sprintf("     Block:      %x", input.Block))
	lines = append(lines, fmt.Sprintf("     TXID:       %x", input.Txid))
	lines = append(lines, fmt.Sprintf("     Index:      %d", input.Index))
	lines = append(lines, fmt.Sprintf("     Signature:  %x", input.Signature))
	lines = append(lines, fmt.Sprintf("     PubKey:     %x", input.PubKey))
	
	if counter > 2 {
		lines = append(lines, fmt.Sprintf("       ."))
		lines = append(lines, fmt.Sprintf("       ."))
		lines = append(lines, fmt.Sprintf("       ."))
	}
	
	if counter > 1 {
		i := counter - 1
		input := tx.Vin[i]
		lines = append(lines, fmt.Sprintf("   Input %d", i))
		lines = append(lines, fmt.Sprintf("     Block:      %x", input.Block))
		lines = append(lines, fmt.Sprintf("     TXID:       %x", input.Txid))
		lines = append(lines, fmt.Sprintf("     Index:      %d", input.Index))
		lines = append(lines, fmt.Sprintf("     Signature:  %x", input.Signature))
		lines = append(lines, fmt.Sprintf("     PubKey:     %x", input.PubKey))
	}
	
	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("   Output %d", i))
		lines = append(lines, fmt.Sprintf("     Value:      %d", output.Value))
		lines = append(lines, fmt.Sprintf("     PubKeyHash: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

func (tx *Transaction) IsCoinbase() bool {
	var emptyArray [32]byte
	return len(tx.Vin) == 1 && tx.Vin[0].Block == emptyArray && tx.Vin[0].Txid == emptyArray && tx.Vin[0].Index == -1 && len(tx.Vin[0].Signature) == 0
}

func (tnx *Transaction) AddNewTxToBucket(db *bolt.DB) error {
	_, err := VerifyTransactionWithoutChain(tnx)
	
	if err != nil {
		return err
	}
	
	txExists := false
	isDoubleSpending := false
	
	err = db.View(func(tx *bolt.Tx) error {
		txs := tx.Bucket([]byte(txsBucket))
		c := txs.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var txid [32]byte
			copy(txid[:], k)
			
			if txid == tnx.ID {
				txExists = true
				break
			}
			
			tranx := DeserializeTransaction(v)
			
			for _, vin := range tranx.Vin {
				for _, tnxv := range tnx.Vin {
					if tnxv.Block == vin.Block && tnxv.Txid == vin.Txid && tnxv.Index == vin.Index {
						isDoubleSpending = true
						break
					}
				}
				
				if isDoubleSpending {
					break
				}
			}
			
			if isDoubleSpending {
				break
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	switch {
	case txExists:
		return errors.New("Error: The transaction already exists or a hash collision has occurred.")
		
	case isDoubleSpending:
		return errors.New("Error: Double Spending")
		
	default:
		err := db.Update(func(tx *bolt.Tx) error {
			txs := tx.Bucket([]byte(txsBucket))
			err := txs.Put(tnx.ID[:], tnx.Serialize())
			CheckErr(err)
			
			return nil
		})
		CheckErr(err)
		
		return nil
	}
}

func (tx *Transaction) TrimmedCopy() *Transaction {
	var inputs []TXInput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Block, vin.Txid, vin.Index, nil, nil})
	}

	txCopy := &Transaction{[32]byte{}, tx.Version, inputs, tx.Vout}

	return txCopy
}

func (tx *Transaction) Sign(wallets *Wallets, bc *Blockchain) {
	if tx.IsCoinbase() {
		return
	}
	
	txCopy := tx.TrimmedCopy()
	
	for inID, vin := range tx.Vin {
		pubKeyHash, err := bc.FindTXOutputPubKeyHash(vin.Block, vin.Txid, vin.Index)
		CheckErr(err)
		
		privKey, err := GetPrivateKey(pubKeyHash, wallets)
		CheckErr(err)
		
		txCopy.Vin[inID].PubKey = pubKeyHash[:]
		dataToSign := sha256.Sum256(txCopy.MarshalJSON())
		txCopy.Vin[inID].PubKey = nil
		
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, dataToSign[:])
		CheckErr(err)
		
		signature := append(r.Bytes(), s.Bytes()...)
		
		tx.Vin[inID].Signature = signature
	}
}

func (tx *Transaction) VerifySign(bc *Blockchain) bool {
	if tx.IsCoinbase() {
		return false
	}
	
	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()
	
	for inID, vin := range tx.Vin {
		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])
		
		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
		
		pubKeyHash, err := bc.FindTXOutputPubKeyHash(vin.Block, vin.Txid, vin.Index)
		CheckErr(err)
		
		txCopy.Vin[inID].PubKey = pubKeyHash[:]
		dataToVerify := sha256.Sum256(txCopy.MarshalJSON())
		txCopy.Vin[inID].PubKey = nil
		
		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])
		
		if ecdsa.Verify(&rawPubKey, dataToVerify[:], &r, &s) == false {
			// str := "Error: " + strconv.Itoa(int(vin.Index)) + "@" + hex.EncodeToString(vin.Txid[:]) + "@" + hex.EncodeToString(vin.Block[:])
			// PrintMessage(str)
			
			return false
		}
	}
	
	return true
}

func (tx *Transaction) Hash() [32]byte {
	txCopy := *tx
	txCopy.ID = [32]byte{}
	
	return sha256.Sum256(txCopy.MarshalJSON())
}

func (tx *Transaction) VerifySize() (uint32, error) {
	txByte := tx.Serialize()
	txSize := uint32(len(txByte))
	
	if txSize < uint32(minTxSize) {
		return txSize, errors.New("Error: The transaction file is too small")
	}
	
	if txSize > maxBlockSize - uint32(maxBlockHeaderPayload) - uint32(coinbaseReservedSize) {
		return txSize, errors.New("Error: The transaction file is too large")
	}
	
	return txSize, nil
}

func (tx *Transaction) MarshalJSON() []byte {
	data_byte, err := json.Marshal(tx)
	CheckErr(err)
	
	return data_byte
}

func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	CheckErr(err)

	return encoded.Bytes()
}

func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	CheckErr(err)

	return transaction
}

func NewCoinbaseTX(to string, txAmount uint, blockHeight uint32) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput
	
	txin := TXInput{[32]byte{}, [32]byte{}, -1, nil, Uint32ToByte(blockHeight)}
	
	inputs = append(inputs, txin)
	outputs = append(outputs, NewTXOutput(Subsidy(blockHeight), []byte(to)))
	
	if txAmount > 0 {
		outputs = append(outputs, NewTXOutput(uint32(txAmount), []byte(to)))
	}
	
	tx := &Transaction{[32]byte{}, transactionVersion, inputs, outputs}
	tx.ID = tx.Hash()

	return tx
}

func NewUTXOTransaction(from, to string, amount uint32, bc *Blockchain) (tx *Transaction, err error) {
	var contractChainBlockHash [20]byte 

	if amount == 0 {
		contractChainBlockHash, err = ContractChainBlockHashDecodeString(to)

		if err != nil {
			return tx, err
		}

	} else {
		if ValidateAddress(to) != true {
			return tx, errors.New("ERROR: The recipient’s address is not valid.")
		}
	}
	
	usableWallets := Wallets{}
	usableWallets.Wallets = make(map[string]*Wallet)
	
	wallets, err := NewWallets(nodeId)
	
	if err != nil {
		return tx, err
	}
	
	if len(from) > 0 {
		if ValidateAddress(from) != true {
			return tx, errors.New("ERROR: The sender’s address is not valid.")
		}
		
		if from == to {
			return tx, errors.New("ERROR: The sender’s address and the recipient’s address cannot be the same.")
		}

		wallet, err := wallets.GetWallet(from)

		if err != nil {
			return tx, err
		}

		usableWallets.Wallets[from] = wallet
		
	} else {
		if amount == 0 {
			return tx, errors.New("ERROR: The sender’s address is not valid.")
		}

		addresses := wallets.GetAddresses()
		
		for _, address := range addresses {
			if address != to {
				wallet, err := wallets.GetWallet(address)
				
				if err != nil {
					return tx, err
				}

				usableWallets.Wallets[address] = wallet
			}
		}
	}
	
	amountAndFee := amount + 1
	
	UTXOSet := UTXOSet{bc}
	
	acc, validOutputs, err := UTXOSet.FindSpendableOutputs(&usableWallets, amountAndFee)
	
	if err != nil {
		return tx, err
	}
	
	var inputs []TXInput
	var outputs []TXOutput
	
	for _, out := range validOutputs.Outputs {
		input := TXInput{out.Block, out.Txid, out.Index, nil, out.PubKey}
		inputs = append(inputs, input)
	}
	
	if amount == 0 {
		outputs = append(outputs, TXOutput{amount, contractChainBlockHash})
	} else {
		outputs = append(outputs, NewTXOutput(amount, []byte(to)))
	}

	if acc > amountAndFee {
		pubKeyHash := HashPubKey(validOutputs.Outputs[0].PubKey)
		address := GetAddress(pubKeyHash)
		outputs = append(outputs, NewTXOutput(acc-amountAndFee, address))
	}
	
	tx = &Transaction{[32]byte{}, transactionVersion, inputs, outputs}
	tx.Sign(&usableWallets, bc)
	tx.ID = tx.Hash()
	
	return tx, nil
}

func VerifyTransactionsWithoutChain(txs []*Transaction) error {
	txAmount := len(txs)
	errMSG := "Transactions verification failed."
	
	if txAmount < 1 {
		return errors.New(errMSG)
	}
	
	if txAmount > int(maxTxAmount) {
		return errors.New(errMSG)
	}
	
	for i := 0; i < txAmount - 1; i++ {
		objA := txs[i].ID
		
		for j := i + 1; j < txAmount; j++ {
			objB := txs[j].ID
			
			if objA == objB {
				return errors.New(errMSG)
			}
		}
	}
	
	if txAmount > 2 {
		for i := 1; i < txAmount - 1; i++ {
			for _, objA := range txs[i].Vin {
				for j := i + 1; j < txAmount; j++ {
					for _, objB := range txs[j].Vin {
						if objA.Block == objB.Block && objA.Txid == objB.Txid && objA.Index == objB.Index {
							return errors.New(errMSG)
						}
					}
				}
			}
		}
	}
	
	var txsSize uint32
	
	for index, tx := range txs {
		var txHashVerify Transaction
		txHashVerify.ID = tx.Hash()
		
		if tx.ID != txHashVerify.ID {
			return errors.New(errMSG)
		}
		
		if tx.Version != transactionVersion {
			return errors.New(errMSG)
		}
		
		if len(tx.Vin) == 0 {
			return errors.New(errMSG)
		}
		
		if len(tx.Vout) == 0 {
			return errors.New(errMSG)
		}
		
		if index == 0 {
			err := VerifyCoinbaseWithoutChain(tx, txAmount)
			
			if err != nil {
				return err
			}
			
		} else {
			object_length := len(tx.Vin)
			
			for i := 0; i < object_length - 1; i++ {
				objA := tx.Vin[i]
				
				for j := i + 1; j < object_length; j++ {
					objB := tx.Vin[j]
					
					if objA.Block == objB.Block && objA.Txid == objB.Txid && objA.Index == objB.Index {
						return errors.New(errMSG)
					}
				}
			}
			
			txSize, err := VerifyTransactionWithoutChain(tx)
			
			if err != nil {
				return err
			}
			
			txsSize += txSize
		}
	}
	
	if txsSize > maxBlockSize - uint32(maxBlockHeaderPayload) - uint32(coinbaseReservedSize) {
		return errors.New(errMSG)
	}
	
	return nil
}

func VerifyCoinbaseWithoutChain(coinbase *Transaction, txAmount int) error {
	errMSG := "Coinbase verification failed."
	
	if coinbase.IsCoinbase() != true {
		return errors.New(errMSG)
	}
	
	coinbaseByte := coinbase.Serialize()
	coinbaseSize := uint(len(coinbaseByte))
	
	if coinbaseSize < uint(minTxSize) || coinbaseSize > uint(coinbaseReservedSize) {
		return errors.New(errMSG)
	}
	
	if txAmount > 1 {
		if len(coinbase.Vout) != 2 {
			return errors.New(errMSG)
		}
		
		if uint32(txAmount - 1) != coinbase.Vout[1].Value {
			return errors.New(errMSG)
		}
		
		if coinbase.Vout[0].PubKeyHash != coinbase.Vout[1].PubKeyHash {
			return errors.New(errMSG)
		}
		
	} else {
		if len(coinbase.Vout) != 1 {
			return errors.New(errMSG)
		}
	}
	
	if coinbase.Vout[0].Value < 1 || coinbase.Vout[0].Value > genesisSubsidy {
		return errors.New(errMSG)
	}
	
	if len(coinbase.Vin[0].PubKey) == 0 {
		return errors.New(errMSG)
	}
	
	blockHeight := ByteToUint32(coinbase.Vin[0].PubKey)
	
	if coinbase.Vout[0].Value != Subsidy(blockHeight) {
		return errors.New(errMSG)
	}
	
	return nil
}

func VerifyTransactionWithoutChain(tx *Transaction) (uint32, error) {
	errMSG := "Transaction verification failed."
	
	txSize, err := tx.VerifySize()
	
	if err != nil {
		return txSize, errors.New(errMSG)
	}
	
	if tx.IsCoinbase() == true {
		return txSize, errors.New(errMSG)
	}
	
	if len(tx.Vin) > int(maxVins) {
		return txSize, errors.New(errMSG)
	}
	
	if tx.Vout[0].Value < 0 {
		return txSize, errors.New(errMSG)
	}

	txVoutLen := len(tx.Vout)
	
	switch txVoutLen {
	case 1:
		for _, vin := range tx.Vin {
			if HashPubKey(vin.PubKey) == tx.Vout[0].PubKeyHash {
				return txSize, errors.New(errMSG)
			}
		}
		
	case 2:
		if tx.Vout[1].Value < 1 {
			return txSize, errors.New(errMSG)
		}

		if tx.Vout[0].PubKeyHash == tx.Vout[1].PubKeyHash {
			return txSize, errors.New(errMSG)
		}
		
		if tx.Vout[1].PubKeyHash != HashPubKey(tx.Vin[0].PubKey) {
			return txSize, errors.New(errMSG)
		}
		
	default:
		return txSize, errors.New(errMSG)
	}
	
	return txSize, nil
}