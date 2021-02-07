package wakacoin

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"
	
	"github.com/boltdb/bolt"
)

type getbestheight struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
}

type bestheight struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
	BestHeight  uint32
	LastHash    [32]byte
}

type getdata struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
	Kind        string
	ID          [32]byte
}

type block struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
	Block       []byte
}

type txidz struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
	Txids       [][32]byte
}

type trx struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
	Transaction []byte
}

func StartServer(nodeID, minerAddress, from, to string, sendNewTx bool, amount uint) {
	addrHost, _, err := net.SplitHostPort(DefaultHub)
	CheckErr(err)
	
	err = ValidateAddrHost(addrHost)
	CheckErr(err)
	
	nodeId = nodeID
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	MiningAddress = minerAddress
	bc := NewBlockchain(nodeID)
	
	blockHashesFromGenesisToTip = *bc.GetBlockHashes()
	ReverseHashes(blockHashesFromGenesisToTip)
	
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
	
	topBlock, err := bc.GetBlock(bc.tip)
	CheckErr(err)
	
	bestHeight := topBlock.GetHeight()
	
	if bestHeight < protectHeight {
		isVerifyBlocks = false
		
	} else {
		fmt.Println("Verify Protect")
		err := bc.VerifyProtect()
		
		if err == nil {
			err := bc.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(paramBucket))
				verify := b.Get([]byte("valid"))
				
				if verify == nil {
					err := b.Put([]byte("valid"), []byte("N"))
					CheckErr(err)
					
					isVerifyBlocks = false
					
				} else {
					if bytes.Compare(verify, []byte("Y")) == 0 {
						isVerifyBlocks = true
						
					} else {
						isVerifyBlocks = false
					}
				}

				return nil
			})
			CheckErr(err)
			
			if isVerifyBlocks == false {
				fmt.Println("Verify Blocks")
				var emptyArray [32]byte
				bci := bc.Iterator()
				
				for {
					block := bci.Next()
					
					if err := VerifyBlockWithoutChain(block); err != nil {
						break
					}
					
					if err := bc.VerifyBlock(block, &blockHashesFromGenesisToTip); err != nil {
						break
					}
					
					if block.Header.PrevBlock == emptyArray {
						err := bc.db.Update(func(tx *bolt.Tx) error {
							b := tx.Bucket([]byte(paramBucket))
							err := b.Put([]byte("valid"), []byte("Y"))
							CheckErr(err)
							
							return nil
						})
						CheckErr(err)
						
						isVerifyBlocks = true
						
						break
					}
				}
			}
			
		} else {
			err := bc.db.Update(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(paramBucket))
				err := b.Put([]byte("valid"), []byte("N"))
				CheckErr(err)
				
				return nil
			})
			CheckErr(err)
			
			isVerifyBlocks = false
			
			if protectHeight > 0 {
				fmt.Println("Trim the blockchain")
				bc.Trim()
			}
		}
	}
	
	c := make(chan bool)
	
	var ln net.Listener
	
	if IsTest {
		ln, err = net.Listen(protocol, nodeAddress)
		
	} else {
		port := fmt.Sprintf(":%s", nodeID)
		ln, err = net.Listen(protocol, port)
	}
	
	CheckErr(err)
	defer ln.Close()
	
	go func() {
		for {
			defer func() {
				if err := recover(); err != nil {
					PrintErr(err)
				}
			}()
			
			conn, err := ln.Accept()
			CheckErr(err)
			go handleConnection(conn, bc)
		}
	}()
	
	if !localhostisDefaultHub {
		go func() {
			for {
				defer func() {
					if err := recover(); err != nil {
						PrintErr(err)
					}
				}()
				
				ConnectDefaultHub(bc.db)
				time.Sleep(30 * time.Minute)
			}
		}()
	}
	
	go func() {
		for {
			defer func() {
				if err := recover(); err != nil {
					PrintErr(err)
				}
			}()
			
			ConnectHubs(bc.db)
			time.Sleep(20 * time.Minute)
		}
	}()
	
	time.Sleep(15 * time.Second)
	
	nodesPacket := []string{}
	nodesPacketMax := 8
	knownNodesLength := len(knownNodes)
	
	if knownNodesLength < nodesPacketMax + 1 {
		for node, _ := range knownNodes {
			nodesPacket = append(nodesPacket, node)
		}
		
	} else {
		randomNumbers := generateRandomNumber(0, knownNodesLength - 1, nodesPacketMax)
		
		index := 0
		
		for node, _ := range knownNodes {
			for _, num := range randomNumbers {
				if index == num {
					nodesPacket = append(nodesPacket, node)
				}
			}
			
			index ++
		}
	}
	
	for _, node := range nodesPacket {
		send := true
		
		switch {
		case node == nodeAddress:
			send = false
			
		case node == DefaultHub:
			if DefaultHubIsIP == false {
				send = false
			}
			
		case SetLocalhostDomainName && node == hostDomainName:
			send = false
			
		case SetLocalhostStaticIPAddr && node == hostStaticAddress:
			send = false
		}
		
		if send {
			if localhostisDefaultHub {
				sendBestHeight(node, bc, false)
				
			} else {
				sendGetBestHeight(node, bc.db, false)
			}
		}
	}
	
	if isVerifyBlocks {
		time.Sleep(10 * time.Second)
		
	} else {
		for {
			time.Sleep(5 * time.Minute)
			
			if isVerifyBlocks {
				break
			}
			
			for k, _ := range knownNodes {
				send := true
				
				switch {
				case k == nodeAddress:
					send = false
					
				case k == DefaultHub:
					if DefaultHubIsIP == false {
						send = false
					}
					
				case SetLocalhostDomainName && k == hostDomainName:
					send = false
					
				case SetLocalhostStaticIPAddr && k == hostStaticAddress:
					send = false
				}
				
				if send {
					sendGetBestHeight(k, bc.db, false)
				}
			}
			
			time.Sleep(5 * time.Minute)
			
			if isVerifyBlocks {
				break
			}
			
			if err := bc.VerifyProtect(); err == nil {
				var emptyArray [32]byte
				bci := bc.Iterator()
				
				for {
					block := bci.Next()
					
					if err := VerifyBlockWithoutChain(block); err != nil {
						break
					}
					
					if err := bc.VerifyBlock(block, &blockHashesFromGenesisToTip); err != nil {
						break
					}
					
					if block.Header.PrevBlock == emptyArray {
						err := bc.db.Update(func(tx *bolt.Tx) error {
							b := tx.Bucket([]byte(paramBucket))
							err := b.Put([]byte("valid"), []byte("Y"))
							CheckErr(err)
							
							return nil
						})
						CheckErr(err)
						
						isVerifyBlocks = true
						
						break
					}
				}
			}
			
			if isVerifyBlocks {
				break
			}
		}
	}
	
	switch {
	case len(MiningAddress) > 0:
		go func() {
			for {
				defer func() {
					if err := recover(); err != nil {
						PrintErr(err)
					}
				}()
				
				time.Sleep(2 * time.Minute)
				SendTXIDs(bc.db)
				time.Sleep(8 * time.Minute)
			}
		}()
		
		for {
			StartMine(bc, 5000000, 0, 0)
			// StartMine(bc, 15000, 900000, 10)
		}
		
	case sendNewTx:
		var tx *Transaction
		var err error
		
		for {
			tx, err = NewUTXOTransaction(from, to, amount, bc)
			CheckErr(err)
			
			fmt.Println("verify sign")
			isValid := tx.VerifySign(bc)
			
			if isValid {
				break
			}
		}
		
		err = tx.AddNewTxToBucket(bc.db)
		
		if err == nil {
			PrintMessage("The transaction was successfully submitted.")
			txSerialize := tx.Serialize()
			
			for k, _ := range knownNodes {
				send := true
				
				switch {
				case k == nodeAddress:
					send = false
					
				case k == DefaultHub:
					if DefaultHubIsIP == false {
						send = false
					}
					
				case SetLocalhostDomainName && k == hostDomainName:
					send = false
					
				case SetLocalhostStaticIPAddr && k == hostStaticAddress:
					send = false
				}
				
				if send {
					sendTx(k, txSerialize, bc.db, true)
				}
			}
			
			time.Sleep(1 * time.Minute)
			os.Exit(1)
			
		} else {
			PrintErr(err)
		}
	}
	
	<-c
}

func handleConnection(conn net.Conn, bc *Blockchain) {
	defer func() {
		if err := recover(); err != nil {
			PrintErr(err)
		}
    }()
	
	remoteAddr := conn.RemoteAddr().String()
	remoteAddrHost, _, err := net.SplitHostPort(remoteAddr)
	CheckErr(err)
	
	err = ValidateAddrHost(remoteAddrHost)
	CheckErr(err)
	
	request, err := ioutil.ReadAll(conn)
	CheckErr(err)
	
	t := time.Now().UTC()
	timeNow := t.Format("02 Jan 15:04:05")
	command := bytesToCommand(request[:CommandLength])
	fmt.Printf("%s Received %s command\n", timeNow, command)
	
	switch command {
	case "gethubs":
		handleGetHubs(remoteAddrHost, request, bc.db)
		
	case "hubs":
		handleHubs(remoteAddrHost, request, bc.db)
		
	case "getknownnodes":
		handleGetKnownNodes(remoteAddrHost, request, bc.db)
		
	case "knownodepacket":
		handleKnownNodesPacket(remoteAddrHost, request)
		
	case "getbestheight":
		handleGetBestHeight(remoteAddrHost, request, bc)
	
	case "bestheight":
		handleBestHeight(remoteAddrHost, request, bc)
		
	case "getdata":
		handleGetData(remoteAddrHost, request, bc)
		
	case "block":
		handleBlock(remoteAddrHost, request, bc.db)
		
	case "txids":
		handleTxIDs(remoteAddrHost, request, bc.db)
		
	case "tx":
		handleTx(remoteAddrHost, request, bc)
		
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

func handleGetBestHeight(remoteAddrHost string, request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getbestheight
	
	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	err = ValidateBlockChain(payload.BlockChain)
	CheckErr(err)
	
	remoteAddr, err := RemoteAddr(payload.NodeVersion ,payload.StaticAddr, payload.Host, payload.Port, remoteAddrHost)
	CheckErr(err)
	
	nodeExists, _ := nodeIsKnown(remoteAddr)
	manageKnownNodes(nodeExists, remoteAddr)
	
	sendBestHeight(remoteAddr, bc, false)
}

func handleBestHeight(remoteAddrHost string, request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload bestheight
	
	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	err = ValidateBlockChain(payload.BlockChain)
	CheckErr(err)
	
	remoteAddr, err := RemoteAddr(payload.NodeVersion ,payload.StaticAddr, payload.Host, payload.Port, remoteAddrHost)
	CheckErr(err)
	
	nodeExists, _ := nodeIsKnown(remoteAddr)
	manageKnownNodes(nodeExists, remoteAddr)
	
	topBlock, err := bc.GetBlock(bc.tip)
	CheckErr(err)
	
	myBestHeight := topBlock.GetHeight()
	
	switch {
	case myBestHeight > payload.BestHeight:
		if isVerifyBlocks {
			sendBestHeight(remoteAddr, bc, false)
		}
		
	case myBestHeight < payload.BestHeight:
		if isVerifying {
			return
		}
		
		nodeIsOnBlacklist := NodeIsOnBlacklist(remoteAddr)
		
		if nodeIsOnBlacklist {
			return
		}
		
		if isVerifyBlocks && ((payload.BestHeight - 1) == myBestHeight) {
			var block *Block
			var encodedBlock []byte
			
			err := bc.db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(blocksBucket))
				encodedBlock = b.Get(payload.LastHash[:])
				
				if encodedBlock != nil {
					block = DeserializeBlock(encodedBlock)
				}

				return nil
			})
			CheckErr(err)
			
			if encodedBlock == nil {
				sendGetData(remoteAddr, "block", payload.LastHash, bc.db, false)
				
				return
			}
			
			if block.Header.PrevBlock == bc.tip {
				if err := VerifyBlockWithoutChain(block); err != nil {
					PutOnBlacklist(remoteAddr)
					
					return
				}
				
				nbc := Blockchain{payload.LastHash, bc.db}
				
				if err := nbc.VerifyBlock(block, &blockHashesFromGenesisToTip); err != nil {
					PutOnBlacklist(remoteAddr)
					
					return
				}
				
				if block.Header.PrevBlock == bc.tip {
					bc.UpdateMainChain(payload.LastHash, block)
					sendGetBestHeight(remoteAddr, bc.db, false)
					
					return
				}
			}
		}
		
		topBlock, err = bc.GetBlock(bc.tip)
		CheckErr(err)
		
		myBestHeight = topBlock.GetHeight()
		
		if myBestHeight == payload.BestHeight {
			return
		}
		
		if myBestHeight > payload.BestHeight {
			sendBestHeight(remoteAddr, bc, false)
			
			return
		}
		
		var emptyArray [32]byte
		bci := &BlockchainIteratorTwo{payload.LastHash, payload.BestHeight, bc.db}
		
		for {
			findNonExistingBlock, findIllegalBlock, blockHash, blockHeight := bci.NextIfBlockExists()
			
			if findNonExistingBlock {
				sendGetData(remoteAddr, "block", blockHash, bc.db, false)
				
				return
			}
			
			if findIllegalBlock {
				PutOnBlacklist(remoteAddr)
				
				return
			}
			
			if isVerifyBlocks {
				if blockHeight == protectHeight {
					slice, err := hex.DecodeString(protectTo)
					CheckErr(err)
					
					var protectToArray [32]byte
					copy(protectToArray[:], slice)
					
					nbc := Blockchain{payload.LastHash, bc.db}
					blockHashes := nbc.GetBlockHashes()
					bci_nbc := nbc.Iterator()
					heightDifference := int(payload.BestHeight - protectHeight)
					
					for i := 0; i < heightDifference; i++ {
						block := bci_nbc.Next()
						
						if i == heightDifference - 1 && block.Header.PrevBlock != protectToArray {
							PutOnBlacklist(remoteAddr)
							
							return
						}
						
						err = nbc.VerifyBlock(block, blockHashes)
						
						if err != nil {
							PutOnBlacklist(remoteAddr)
							
							return
						}
						
						topBlock, err = bc.GetBlock(bc.tip)
						CheckErr(err)
						
						myBestHeight = topBlock.GetHeight()
						
						if myBestHeight == payload.BestHeight {
							return
						}
						
						if myBestHeight > payload.BestHeight {
							sendBestHeight(remoteAddr, bc, false)
							
							return
						}
						
						if i == heightDifference - 1 {
							bc.ChangeMainChain(payload.BestHeight, payload.LastHash)
							sendGetBestHeight(remoteAddr, bc.db, false)
							
							return
						}
					}
					
					return
				}
				
			} else {
				if blockHash == emptyArray {
					nbc := Blockchain{payload.LastHash, bc.db}
					err := nbc.VerifyProtect()
					
					if err != nil {
						PutOnBlacklist(remoteAddr)
						
						return
						
					} else {
						blockHashes := nbc.GetBlockHashes()
						bci_nbc := nbc.Iterator()
						
						for {
							block := bci_nbc.Next()
							
							err := nbc.VerifyBlock(block, blockHashes)
							
							if err != nil {
								PutOnBlacklist(remoteAddr)
								
								return
							}
							
							topBlock, err = bc.GetBlock(bc.tip)
							CheckErr(err)
							
							myBestHeight = topBlock.GetHeight()
							
							if myBestHeight == payload.BestHeight {
								return
							}
							
							if myBestHeight > payload.BestHeight {
								sendBestHeight(remoteAddr, bc, false)
								
								return
							}
							
							if block.Header.PrevBlock == emptyArray {
								bc.ChangeMainChain(payload.BestHeight, payload.LastHash)
								sendGetBestHeight(remoteAddr, bc.db, false)
								
								err := bc.db.Update(func(tx *bolt.Tx) error {
									b := tx.Bucket([]byte(paramBucket))
									err := b.Put([]byte("valid"), []byte("Y"))
									CheckErr(err)
									
									return nil
								})
								CheckErr(err)
								
								isVerifyBlocks = true
								
								return
							}
						}
					}
				}
			}
		}
	}
}

func handleGetData(remoteAddrHost string, request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getdata
	
	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	err = ValidateBlockChain(payload.BlockChain)
	CheckErr(err)
	
	remoteAddr, err := RemoteAddr(payload.NodeVersion ,payload.StaticAddr, payload.Host, payload.Port, remoteAddrHost)
	CheckErr(err)
	
	nodeExists, _ := nodeIsKnown(remoteAddr)
	manageKnownNodes(nodeExists, remoteAddr)
	
	switch payload.Kind {
	case "block":
		block, err := bc.GetBlock(payload.ID)
		
		if err == nil {
			err := VerifyBlockWithoutChain(block)
			
			if err == nil {
				sendBlock(remoteAddr, block, bc.db, false)
			}
		}
		
	case "tx":
		var tnx []byte
		
		err := bc.db.View(func(tx *bolt.Tx) error {
			txs := tx.Bucket([]byte(txsBucket))
			tnx = txs.Get(payload.ID[:])
			
			return nil
		})
		CheckErr(err)
		
		if tnx != nil {
			sendTx(remoteAddr, tnx, bc.db, false)
		}
	}
}

func handleBlock(remoteAddrHost string, request []byte, db *bolt.DB) {
	var buff bytes.Buffer
	var payload block
	
	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	err = ValidateBlockChain(payload.BlockChain)
	CheckErr(err)
	
	remoteAddr, err := RemoteAddr(payload.NodeVersion ,payload.StaticAddr, payload.Host, payload.Port, remoteAddrHost)
	CheckErr(err)
	
	nodeExists, _ := nodeIsKnown(remoteAddr)
	manageKnownNodes(nodeExists, remoteAddr)
	
	b := DeserializeBlock(payload.Block)
	b.AddBlock(db)
	
	err = VerifyBlockWithoutChain(b)
	
	if err == nil {
		sendGetBestHeight(remoteAddr, db, false)
	}
}

func handleTxIDs(remoteAddrHost string, request []byte, db *bolt.DB) {
	var buff bytes.Buffer
	var payload txidz
	
	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	err = ValidateBlockChain(payload.BlockChain)
	CheckErr(err)
	
	remoteAddr, err := RemoteAddr(payload.NodeVersion ,payload.StaticAddr, payload.Host, payload.Port, remoteAddrHost)
	CheckErr(err)
	
	nodeExists, _ := nodeIsKnown(remoteAddr)
	manageKnownNodes(nodeExists, remoteAddr)
	
	var tnxs [][32]byte
	
	err = db.View(func(tx *bolt.Tx) error {
		txs := tx.Bucket([]byte(txsBucket))
		
		txidsPacketMaxInt := int(txidsPacketMax)
		
		for i, txid := range payload.Txids {
			if i < txidsPacketMaxInt {
				t := txs.Get(txid[:])
				
				if t == nil {
					tnxs = append(tnxs, txid)
				}
				
			} else {
				break
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	for _, txid := range tnxs {
		sendGetData(remoteAddr, "tx", txid, db, false)
	}
}

func handleTx(remoteAddrHost string, request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload trx
	
	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	err = ValidateBlockChain(payload.BlockChain)
	CheckErr(err)
	
	remoteAddr, err := RemoteAddr(payload.NodeVersion ,payload.StaticAddr, payload.Host, payload.Port, remoteAddrHost)
	CheckErr(err)
	
	nodeExists, _ := nodeIsKnown(remoteAddr)
	manageKnownNodes(nodeExists, remoteAddr)
	
	tnx := DeserializeTransaction(payload.Transaction)
	
	UTXOSet := UTXOSet{bc}
	
	isSpendable, err := UTXOSet.isSpendableTX(&tnx)
	
	isOK := false
	
	if isSpendable {
		isOK = true
		
	} else {
		if err.Error() == confirmationsError {
			isOK = true
			
		} else {
			fmt.Println(err)
		}
	}
	
	if isOK {
		err := tnx.AddNewTxToBucket(bc.db)
		
		if err == nil {
			var txidsPacket [][32]byte
			
			txidsPacket = append(txidsPacket, tnx.ID)
			
			err := bc.db.View(func(tx *bolt.Tx) error {
				txs := tx.Bucket([]byte(txsBucket))
				c := txs.Cursor()
				
				txidsPacketMaxInt := int(txidsPacketMax)
				
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					var array [32]byte
					copy(array[:], k)
					
					if array != tnx.ID {
						if len(txidsPacket) < txidsPacketMaxInt {
							txidsPacket = append(txidsPacket, array)
							
						} else {
							break
						}
					}
				}
				
				return nil
			})
			CheckErr(err)
			
			nodesPacket := []string{}
			nodesPacketMax := 5
			knownNodesLength := len(knownNodes)
			
			if knownNodesLength < nodesPacketMax + 1 {
				for node, _ := range knownNodes {
					nodesPacket = append(nodesPacket, node)
				}
				
			} else {
				randomNumbers := generateRandomNumber(0, knownNodesLength - 1, nodesPacketMax)
				
				index := 0
				
				for node, _ := range knownNodes {
					for _, num := range randomNumbers {
						if index == num {
							nodesPacket = append(nodesPacket, node)
						}
					}
					
					index ++
				}
			}
			
			for _, node := range nodesPacket {
				send := true
				
				switch {
				case node == nodeAddress:
					send = false
					
				case node == DefaultHub:
					if DefaultHubIsIP == false {
						send = false
					}
					
				case SetLocalhostDomainName && node == hostDomainName:
					send = false
					
				case SetLocalhostStaticIPAddr && node == hostStaticAddress:
					send = false
				}
				
				if send {
					sendTxids(node, txidsPacket, bc.db, false)
				}
			}
		}
	}
}

func sendGetBestHeight(addr string, db *bolt.DB, report bool) {
	data := getbestheight{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId}
	payload := gobEncode(data)
	request := append(commandToBytes("getbestheight"), payload...)
	sendData(addr, &request, db, report)
}

func sendBestHeight(addr string, bc *Blockchain, report bool) {
	topBlock, err := bc.GetBlock(bc.tip)
	CheckErr(err)

	bestHeight := topBlock.GetHeight()
	
	data := bestheight{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId, bestHeight, bc.tip}
	payload := gobEncode(data)
	request := append(commandToBytes("bestheight"), payload...)
	sendData(addr, &request, bc.db, report)
}

func sendGetData(addr, kind string, id [32]byte, db *bolt.DB, report bool) {
	data := getdata{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId, kind, id}
	payload := gobEncode(data)
	request := append(commandToBytes("getdata"), payload...)
	sendData(addr, &request, db, report)
}

func sendBlock(addr string, b *Block, db *bolt.DB, report bool) {
	data := block{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)
	sendData(addr, &request, db, report)
}

func sendTxids(addr string, tnxs [][32]byte, db *bolt.DB, report bool) {
	data := txidz{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId, tnxs}
	payload := gobEncode(data)
	request := append(commandToBytes("txids"), payload...)
	sendData(addr, &request, db, report)
}

func sendTx(addr string, t []byte, db *bolt.DB, report bool) {
	data := trx{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId, t}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)
	sendData(addr, &request, db, report)
}

func sendData(addr string, data *[]byte, db *bolt.DB, report bool) {
	conn, err := net.Dial(protocol, addr)
	
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		ExcludeNode(addr, db)
		
		return
	}
	
	defer conn.Close()
	
	_, err = io.Copy(conn, bytes.NewReader(*data))
	CheckErr(err)
	
	if report {
		str := "Data successfully sent to " + addr
		PrintMessage(str)
	}
}

func commandToBytes(command string) []byte {
	var bytes [CommandLength]byte
	
	for i, c := range command {
		bytes[i] = byte(c)
	}
	
	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte
	
	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}
	
	return fmt.Sprintf("%s", command)
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	CheckErr(err)

	return buff.Bytes()
}

func ExcludeNode(addr string, db *bolt.DB) {
	err := db.Update(func(tx *bolt.Tx) error {
		h := tx.Bucket([]byte(hubsBucket))
		err := h.Delete([]byte(addr))
		CheckErr(err)
		
		return nil
	})
	CheckErr(err)
	
	delete(knownNodes, addr)
}

func PutOnBlacklist(addr string) {
	blacklist[addr] = time.Now().UTC().Unix()
}

func NodeIsOnBlacklist(addr string) bool {
	timeSet := time.Now().UTC().Add(-24 * time.Hour).Unix()
	
	for node, time := range blacklist {
		if node == addr {
			if time > timeSet {
				return true
				
			} else {
				delete(blacklist, addr)
				return false
			}
		}
		
		if time <= timeSet {
			delete(blacklist, node)
		}
	}

	return false
}

func ValidateAddrHost(addr string) error {
	errMSG := "ERROR: " + addr + " is not a valid textual representation of an IP address."
	
	ip := net.ParseIP(addr)
	
	if ip == nil {
		if DefaultHubIsIP == false {
			addrHost, _, err := net.SplitHostPort(DefaultHub)
			CheckErr(err)
			
			if addr != addrHost {
				return errors.New(errMSG)
			}
			
		} else {
			return errors.New(errMSG)
		}
	}
	
	if IsTest {
		if strings.Contains(addr, "localhost") || strings.Contains(addr, "127.0.0.1") || strings.Contains(addr, "::1") {
			return nil
			
		} else {
			return errors.New(errMSG)
		}
		
	} else {
		if strings.Contains(addr, "localhost") || strings.Contains(addr, "127.0.0.1") || strings.Contains(addr, "::1") {
			return errors.New(errMSG)
			
		} else {
			return nil
		}
	}
}

func ValidateBlockChain(blockchain string) error {
	if blockchain == blockChain {
		return nil
	} else {
		return errors.New("Invalid block chain")
	}
}

func RemoteAddr(version uint32, isStaticAddr bool, staticHost, port, remoteAddrHost string) (string, error) {
	err := ValidateAddrHost(remoteAddrHost)
	CheckErr(err)
	
	remoteAddr := remoteAddrHost + ":" + port
	
	if remoteAddr == DefaultHub && nodeVersion < version {
		return remoteAddr, errors.New(upgradeNotice)
	}
	
	if isStaticAddr {
		err := ValidateAddrHost(staticHost)
		CheckErr(err)
		
		remoteAddr = staticHost + ":" + port
	}
	
	return remoteAddr, nil
}

func nodeIsKnown(addr string) (bool, uint8) {
	for node, counter := range knownNodes {
		if node == addr {
			return true, uint8(counter)
		}
	}

	return false, 0
}

func manageKnownNodes(nodeExists bool, addr string) {
	if !nodeExists {
		addrHost, _, err := net.SplitHostPort(addr)
		CheckErr(err)
		
		err = ValidateAddrHost(addrHost)
		CheckErr(err)
		
		knownNodes[addr] = 1
	}
}

func SendTXIDs(db *bolt.DB) {
	var tmp, txidsPacket [][32]byte
	
	err := db.View(func(tx *bolt.Tx) error {
		txs := tx.Bucket([]byte(txsBucket))
		c := txs.Cursor()
		
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			var array [32]byte
			copy(array[:], k)
			
			tmp = append(tmp, array)
		}
		
		return nil
	})
	CheckErr(err)
	
	tmpLength := len(tmp)
	txidsPacketMaxInt := int(txidsPacketMax)
	
	if tmpLength < txidsPacketMaxInt + 1 {
		for _, v := range tmp {
			txidsPacket = append(txidsPacket, v)
		}
		
	} else {
		randomNumbers := generateRandomNumber(0, tmpLength - 1, txidsPacketMaxInt)
		
		for _, num := range randomNumbers {
			txidsPacket = append(txidsPacket, tmp[num])
		}
	}
	
	if len(txidsPacket) > 0 {
		nodesPacket := []string{}
		nodesPacketMax := 5
		knownNodesLength := len(knownNodes)
		
		if knownNodesLength < nodesPacketMax + 1 {
			for node, _ := range knownNodes {
				nodesPacket = append(nodesPacket, node)
			}
			
		} else {
			randomNumbers := generateRandomNumber(0, knownNodesLength - 1, nodesPacketMax)
			
			index := 0
			
			for node, _ := range knownNodes {
				for _, num := range randomNumbers {
					if index == num {
						nodesPacket = append(nodesPacket, node)
					}
				}
				
				index ++
			}
		}
		
		for _, node := range nodesPacket {
			str := "knownNode: " + node
			PrintMessage(str)
			
			send := true
			
			switch {
			case node == nodeAddress:
				send = false
				
			case node == DefaultHub:
				if DefaultHubIsIP == false {
					send = false
				}
				
			case SetLocalhostDomainName && node == hostDomainName:
				send = false
				
			case SetLocalhostStaticIPAddr && node == hostStaticAddress:
				send = false
			}
			
			if send {
				sendTxids(node, txidsPacket, db, true)
			}
		}
	}
}

func StartMine(bc *Blockchain, batch uint32, maxNonce uint64, sleep uint8) {
	if len(MiningAddress) == 0 {
		PrintMessage("ERROR: Miner Address is not found")
		os.Exit(1)
	}
	
	for {
		if !isVerifying {
			break
		}
		
		PrintMessage("Suspend mining and wait for block verification to complete")
		time.Sleep(1 * time.Minute)
	}
	
	defer func() {
		if err := recover(); err != nil {
			PrintErr(err)
		}
	}()
	
	updateUTXO := false
	
	txs := wrappedTXs(bc)
	
	newBlock, err := bc.MineBlock(txs, batch, maxNonce, sleep)
	
	if err != nil {
		PrintErr(err)
		
	} else {
		PrintMessage("New block is mined")
		
		blockHash := newBlock.Header.HashBlockHeader()
		
		err := bc.VerifyBlockHashCollision(blockHash)
		
		if err != nil {
			PrintMessage("ERROR: The newly mined block is discarded because of hash collision")
			
		} else {
			err := bc.AddLastBlock(newBlock)
			
			if err != nil {
				PrintErr(err)
				
			} else {
				updateUTXO = true
				
				str := fmt.Sprintf("Added block %x", blockHash)
				PrintMessage(str)
				
				if miningPool {
					SendConfirmedBlockInfo(WebServerLanAddress, bc)
				}
			}
		}
	}
	
	nodesPacket := []string{}
	nodesPacketMax := 8
	knownNodesLength := len(knownNodes)
	
	if knownNodesLength < nodesPacketMax + 1 {
		for node, _ := range knownNodes {
			nodesPacket = append(nodesPacket, node)
		}
		
	} else {
		randomNumbers := generateRandomNumber(0, knownNodesLength - 1, nodesPacketMax)
		
		index := 0
		
		for node, _ := range knownNodes {
			for _, num := range randomNumbers {
				if index == num {
					nodesPacket = append(nodesPacket, node)
				}
			}
			
			index ++
		}
	}
	
	for _, node := range nodesPacket {
		str := "knownNode: " + node
		PrintMessage(str)
		
		send := true
		
		switch {
		case node == nodeAddress:
			send = false
			
		case node == DefaultHub && DefaultHubIsIP == false:
			send = false
			
		case SetLocalhostDomainName && node == hostDomainName:
			send = false
			
		case SetLocalhostStaticIPAddr && node == hostStaticAddress:
			send = false
		}
		
		if send {
			sendBestHeight(node, bc, true)
		}
	}
	
	if updateUTXO {
		UTXOSet := UTXOSet{bc}
		err := UTXOSet.UpdateLastBlock()
		
		switch {
		case err == nil:
			_ = UTXOSet.UpdateLastBlock()
			PrintMessage("Update UTXO completed")
			
		case err.Error() == utxoHaveToBeReindexed:
			_ = UTXOSet.Reindex()
			_ = UTXOSet.Reindex()
		}
	}
}

func wrappedTXs(bc *Blockchain) []*Transaction {
	var txids [][]byte
	
	err := bc.db.View(func(tx *bolt.Tx) error {
		txsB := tx.Bucket([]byte(txsBucket))
		c := txsB.Cursor()
		
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			txids = append(txids, k)
		}
		
		return nil
	})
	CheckErr(err)
	
	var txs []*Transaction
	UTXOSet := UTXOSet{bc}
	quota := maxBlockSize - uint32(maxBlockHeaderPayload) - uint32(coinbaseReservedSize)
	
	for _, v := range txids {
		var tnx *Transaction
		
		err := bc.db.View(func(tx *bolt.Tx) error {
			txsB := tx.Bucket([]byte(txsBucket))
			t := txsB.Get(v)
			transaction := DeserializeTransaction(t)
			tnx = &transaction
			
			return nil
		})
		CheckErr(err)
		
		isSpendable, err := UTXOSet.isSpendableTX(tnx)
		
		switch {
		case isSpendable:
			txSize, err := tnx.VerifySize()
			
			if err != nil {
				err := bc.db.Update(func(tx *bolt.Tx) error {
					txsB := tx.Bucket([]byte(txsBucket))
					err := txsB.Delete(tnx.ID[:])
					CheckErr(err)
					
					return nil
				})
				CheckErr(err)
				
			} else {
				if quota >= txSize {
					quota -= txSize
					txs = append(txs, tnx)
				}
			}
			
		case !isSpendable && err.Error() != confirmationsError :
			err := bc.db.Update(func(tx *bolt.Tx) error {
				txsB := tx.Bucket([]byte(txsBucket))
				err := txsB.Delete(tnx.ID[:])
				CheckErr(err)
				
				return nil
			})
			CheckErr(err)
		}
		
		if quota < 200 {
			return txs
		}
		
		if len(txs) >= int(maxTxAmount - 1) {
			return txs
		}
	}
	
	return txs
}