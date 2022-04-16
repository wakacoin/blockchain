package wakacoin

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"golang.org/x/crypto/ripemd160"
)

type ContractBlock struct {
	Version   uint32
	PrevBlock [20]byte
	Height    uint32
	Timestamp int64
	FileName  []byte
	File      []byte
}

type ContractChain struct {
	tip [20]byte
	db  *bolt.DB
}

type ContractChainIterator struct {
	currentHash [20]byte
	db          *bolt.DB
}

func (i *ContractChainIterator) Next() (block *ContractBlock) {
	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash[:])
		
		if encodedBlock == nil {
			return errors.New("ERROR: The block does not exist.")
		}
		
		block = DeserializeContractBlock(encodedBlock)

		return nil
	})
	CliCheckErr(err)
	
	i.currentHash = block.PrevBlock
	return
}

func (cc *ContractChain) Iterator() *ContractChainIterator {
	return &ContractChainIterator{cc.tip, cc.db}
}

func (cli *CLI) contractChainCreate(address, filename, nodeID string) {
	if ValidateAddress(address) != true {
		fmt.Println("\n", "ERROR: The address is not valid.")
		os.Exit(1)
	}

	wallets, err := NewWallets(nodeID)
	CliCheckErr(err)

	wallet, err := wallets.GetWallet(address)
	CliCheckErr(err)

	if _, balanceSpendable := GetBalance(address, nodeID); balanceSpendable < 1 {
		fmt.Println("\n", "ERROR: Since the address has zero spendable wakacoin, you cannot pay the transaction fee.")
		os.Exit(1)
	}

	register, err := HasBeenRegisteredOnBlockchain(nodeID, wallet)
	CliCheckErr(err)

	if register {
		fmt.Println("\n", "ERROR: A contract chain created by this address already exists on the Wakacoin blockchain. Each address can only create a unique contract chain.")
		os.Exit(1)
	}

	dbFile := fmt.Sprintf(contractDbFile, address)

	if dbExists(dbFile) {
		errMSG := "ERROR: " + dbFile + " could not be created because the file already exists."
		
		fmt.Println("\n", errMSG)
		os.Exit(1)
	}

	_, err = os.Stat(filename)

	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("\n", "ERROR: The file does not exist.")
		} else {
			fmt.Println("\n", "ERROR: There is an error when reading the file.")
		}

		os.Exit(1)
	}

	fileContent, err := ioutil.ReadFile(filename)
	CliCheckErr(err)

	blockTime := time.Now().UTC().Unix()
	contractBlock := &ContractBlock{contractVersion, [20]byte{}, 0, blockTime, []byte(filename), fileContent}
	contractBlockHash := contractBlock.HashBlock()

	pubKeyHash := HashPubKey(wallet.PublicKey)
	pubKeyHashSlice := pubKeyHash[:]

	for {
		if bytes.Compare(contractBlockHash[:], pubKeyHashSlice) != 0 {
			break
		}

		blockTime++
		contractBlock.Timestamp = blockTime
		contractBlockHash = contractBlock.HashBlock()
	}
	
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	CliCheckErr(err)

	defer db.Close()
	
	err = db.Update(func(tx *bolt.Tx) error {
		parametersB, err := tx.CreateBucketIfNotExists([]byte(paramBucket))
		CliCheckErr(err)
		
		blocksB, err := tx.CreateBucketIfNotExists([]byte(blocksBucket))
		CliCheckErr(err)

		err = blocksB.Put(contractBlockHash[:], contractBlock.Serialize())
		CliCheckErr(err)

		err = parametersB.Put([]byte("lastBlock"), contractBlockHash[:])
		CliCheckErr(err)

		return nil
	})
	CliCheckErr(err)

	PrintMessage("Update blockchain, please wait 10 minutes.")
	
	var minerAddress string
	sendNewTx := true
	contractBlockHashString := fmt.Sprintf("%x", contractBlockHash)

	fmt.Println("\nContract Block Hash String:")
	fmt.Println(contractBlockHashString)
	
	StartServer(nodeID, minerAddress, address, contractBlockHashString, sendNewTx, 0)
}

func (cli *CLI) contractChainValidate(address, nodeID string) {
	errMSG := "Warning! The Contract Chain is not valid."

	cc := NewContractChain(address)
	defer cc.db.Close()

	var emptyArray [20]byte
	cci := cc.Iterator()
	
	var counter uint
	var register bool
	var registered string
	var height uint32
	var highLevelBlockTime int64
	var contractChainBlockHashs, registeredBlockHashs [][20]byte

	for {
		blockHash := cci.currentHash
		block := cci.Next()

		if block.Version != contractVersion || block.Timestamp < 1649692800 || block.Timestamp > time.Now().UTC().Unix() || len(block.FileName) == 0 || len(block.File) == 0 {
			fmt.Println("\n", errMSG)
			os.Exit(1)
		}

		if counter == 0 {
			height = block.Height
			highLevelBlockTime = block.Timestamp

		} else {
			height--

			if height != block.Height {
				fmt.Println("\n", errMSG)
				os.Exit(1)
			}

			if highLevelBlockTime <= block.Timestamp {
				fmt.Println("\n", errMSG)
				os.Exit(1)
			} else {
				highLevelBlockTime = block.Timestamp
			}
		}

		counter++

		if height == 0 {
			if block.PrevBlock != emptyArray {
				fmt.Println("\n", errMSG)
				os.Exit(1)
			}
		}

		if block.HashBlock() != blockHash {
			fmt.Println("\n", errMSG)
			os.Exit(1)
		} else {
			contractChainBlockHashs = append(contractChainBlockHashs, blockHash)
		}

		if block.PrevBlock == emptyArray {
			break
		}
	}

	bc := NewBlockchain(nodeID)
	defer bc.db.Close()

	u := UTXOSet{bc}
	_, contractBucket := u.GetAvailableUTXO()

	err := bc.db.View(func(tx *bolt.Tx) error {
		var array [20]byte
		addressByte := []byte(address)

		b := tx.Bucket(contractBucket)
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			copy(array[:], k[:20])
			addr := GetAddress(array)

			if bytes.Compare(addr, addressByte) == 0 {
				register = false
				copy(array[:], k[20:])

				for _, blockHash := range contractChainBlockHashs {
					if array == blockHash {
						register = true
						registeredBlockHashs = append(registeredBlockHashs, array)
						break
					}
				}

				if register == false {
					fmt.Println("\n", errMSG)
					os.Exit(1)
				}
			}
		}

		return nil
	})
	CliCheckErr(err)

	Reverse20ByteHashes(contractChainBlockHashs)
	counter = 0

	for _, blockHash := range contractChainBlockHashs {
		register = false

		for _, registeredBlockHash := range registeredBlockHashs {
			if blockHash == registeredBlockHash {
				register = true
				break
			}
		}

		if counter == 0 {
			if register == false {
				fmt.Println("\n", errMSG)
				os.Exit(1)
			}
		}

		registered = "No"

		if register {
			registered = "Yes"
		}

		fmt.Println("\nBlock Height: ", counter)
		fmt.Println("Has this block been registered on the WakaCoin blockchain? ", registered)
		
		counter++
	}

	os.Exit(1)
}

func (cli *CLI) contractCreate(address, filename, nodeID string) {
	cc := NewContractChain(address)
	defer cc.db.Close()

	wallets, err := NewWallets(nodeID)
	CliCheckErr(err)

	wallet, err := wallets.GetWallet(address)
	CliCheckErr(err)

	if _, balanceSpendable := GetBalance(address, nodeID); balanceSpendable < 1 {
		fmt.Println("\n", "ERROR: Since the address has zero spendable wakacoin, you cannot pay the transaction fee.")
		os.Exit(1)
	}

	register, err := HasBeenRegisteredOnBlockchain(nodeID, wallet)
	CliCheckErr(err)

	if !register {
		fmt.Println("\n", "ERROR: The contract chain created by this address cannot be found in the Wakacoin blockchain. Please create the contract chain first.")
		os.Exit(1)
	}

	_, err = os.Stat(filename)

	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("\n", "ERROR: The file does not exist.")
		} else {
			fmt.Println("\n", "ERROR: There is an error when reading the file.")
		}

		os.Exit(1)
	}

	fileContent, err := ioutil.ReadFile(filename)
	CliCheckErr(err)

	var contractBlockHash [20]byte
	
	err = cc.db.Update(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		blocksB := tx.Bucket([]byte(blocksBucket))

		lastBlockByte := blocksB.Get(cc.tip[:])
		lastBlock := DeserializeContractBlock(lastBlockByte)
		
		blockTime := time.Now().UTC().Unix()
		height := lastBlock.Height + 1
		contractBlock := &ContractBlock{contractVersion, cc.tip, height, blockTime, []byte(filename), fileContent}
		contractBlockHash = contractBlock.HashBlock()

		pubKeyHash := HashPubKey(wallet.PublicKey)
		pubKeyHashSlice := pubKeyHash[:]

		for {
			if bytes.Compare(contractBlockHash[:], pubKeyHashSlice) != 0 {
				break
			}

			blockTime++
			contractBlock.Timestamp = blockTime
			contractBlockHash = contractBlock.HashBlock()
		}

		err = blocksB.Put(contractBlockHash[:], contractBlock.Serialize())
		CliCheckErr(err)

		err = parametersB.Put([]byte("lastBlock"), contractBlockHash[:])
		CliCheckErr(err)
		
		return nil
	})
	CliCheckErr(err)

	PrintMessage("Update blockchain, please wait 10 minutes.")
	
	var minerAddress string
	sendNewTx := true
	contractBlockHashString := fmt.Sprintf("%x", contractBlockHash)

	fmt.Println("\nContract Block Hash String:")
	fmt.Println(contractBlockHashString)
	
	StartServer(nodeID, minerAddress, address, contractBlockHashString, sendNewTx, 0)
}

func (cli *CLI) contractListFiles(address string) {
	cc := NewContractChain(address)
	defer cc.db.Close()

	var emptyArray [20]byte
	cci := cc.Iterator()
	
	for {
		block := cci.Next()

		fmt.Println("\nBlock Height: ", block.Height)
		fmt.Println("FileName: ", string(block.FileName))

		if block.PrevBlock == emptyArray {
			break
		}
	}

	os.Exit(1)
}

func (cli *CLI) contractReleaseFile(address string, height uint) {
	cc := NewContractChain(address)
	defer cc.db.Close()

	cci := cc.Iterator()
	
	for {
		block := cci.Next()

		if uint(block.Height) < height {
			break
		}

		if uint(block.Height) == height {
			err := ioutil.WriteFile(string(block.FileName), block.File, 0644)
			CliCheckErr(err)

			break
		}
	}

	os.Exit(1)
}

func NewContractChain(address string) *ContractChain {
	if ValidateAddress(address) != true {
		fmt.Println("\n", "ERROR: The address is not valid.")
		os.Exit(1)
	}

	dbFile := fmt.Sprintf(contractDbFile, address)

	if !dbExists(dbFile) {
		errMSG := "ERROR: Require the database file - " + dbFile
		
		fmt.Println("\n", errMSG)
		os.Exit(1)
	}

	var lastBlockHash []byte

	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	CliCheckErr(err)

	err = db.View(func(tx *bolt.Tx) error {
		parametersB := tx.Bucket([]byte(paramBucket))
		lastBlockHash = parametersB.Get([]byte("lastBlock"))
		
		if lastBlockHash == nil {
			errMSG := "ERROR: No existing last block hash found."
			
			fmt.Println("\n", errMSG)
			os.Exit(1)
		}

		return nil
	})
	CliCheckErr(err)

	var tip [20]byte
	copy(tip[:], lastBlockHash)
	
	return &ContractChain{tip, db}
}

func (b *ContractBlock) HashBlock() [20]byte {
	data := bytes.Join(
		[][]byte{
			Uint32ToByte(b.Version),
			b.PrevBlock[:],
			Uint32ToByte(b.Height),
			Int64ToByte(b.Timestamp),
			b.FileName,
			b.File,
		},
		[]byte{},
	)	
	
	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(data)
	CliCheckErr(err)
	dataRIPEMD160 := RIPEMD160Hasher.Sum(nil)
	
	var array [20]byte
	copy(array[:], dataRIPEMD160)
	return array
}

func (b *ContractBlock) Serialize() []byte {
	var result bytes.Buffer
	
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(b)
	CliCheckErr(err)

	return result.Bytes()
}

func DeserializeContractBlock(d []byte) *ContractBlock {
	var block ContractBlock

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	CheckErr(err)

	return &block
}

func HasBeenRegisteredOnBlockchain(nodeID string, wallet *Wallet) (register bool, err error) {
	bc := NewBlockchain(nodeID)
	defer bc.db.Close()

	u := UTXOSet{bc}
	_, contractBucket := u.GetAvailableUTXO()
	
	err = bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(txsBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			tnx := DeserializeTransaction(v)

			if tnx.Vout[0].Value == 0 {
				if bytes.Compare(tnx.Vin[0].PubKey, wallet.PublicKey) == 0 {
					register = true
					break
				}
			}
		}

		if register {
			return nil
		}

		b = tx.Bucket(contractBucket)
		c = b.Cursor()

		pubKeyHash := HashPubKey(wallet.PublicKey)
		pubKeyHashSlice := pubKeyHash[:]

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if bytes.Compare(k[:20], pubKeyHashSlice) == 0 {
				register = true
				break
			}
		}

		return nil
	})
	CliCheckErr(err)
	
	return register, nil
}

func ContractChainBlockHashDecodeString(contractChainBlockHashString string) (contractChainBlockHash [20]byte, err error) {
	if len(contractChainBlockHashString) != 40 {
		return contractChainBlockHash, errors.New("ERROR: The Contract Chain Block Hash String is not valid. The string length is not equal to 40.")
	}
	
	hashByte, err := hex.DecodeString(contractChainBlockHashString)

	if err != nil {
		return contractChainBlockHash, errors.New("ERROR: The Contract Chain Block Hash String is not valid.")
	}
	
	copy(contractChainBlockHash[:], hashByte)
	return contractChainBlockHash, nil
}
