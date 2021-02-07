package wakacoin

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strconv"
	"time"
	
	"each1"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/boltdb/bolt"
)

type miningjob struct {
	AddrFrom    string
	PrepareData []byte
	Difficulty  uint8
}

type nonze struct {
	AddrFrom string
	Nonce    uint64
}

type confirmedblockinfo struct {
	AddrFrom  string
	Height    uint32
}

func SendAdminEmail(message string) {
	// 除了 server_mining_pool.go 以外的區塊鏈檔案是要開源的，不方便引入 each1 包，所以需要做一個中介
	
	each1.SendAdminEmail(message)
}

func StartMiningPool(nodeID string) {
	addrHost, _, err := net.SplitHostPort(DefaultHub)
	CheckErr(err)
	
	err = ValidateAddrHost(addrHost)
	CheckErr(err)
	
	nodeId = nodeID
	miningPool = true
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	bc := NewBlockchain(nodeID)
	
	blockHashesFromGenesisToTip = *bc.GetBlockHashes()
	ReverseHashes(blockHashesFromGenesisToTip)
	fmt.Printf("Genesis Block is %x \n", blockHashesFromGenesisToTip[0])
	fmt.Printf("protectTo Block is %x \n", blockHashesFromGenesisToTip[protectHeight])
	fmt.Printf("Top Block is %x \n", blockHashesFromGenesisToTip[(len(blockHashesFromGenesisToTip)-1)])
	
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
				fmt.Println("Trim Block Chain")
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
	
	lanPort, err := strconv.Atoi(nodeID)
	CheckErr(err)
	lanPort += 1
	nodeLanAddress = fmt.Sprintf("localhost:%d", lanPort)
	
	lan, err := net.Listen(protocol, nodeLanAddress)
	CheckErr(err)
	
	defer lan.Close()
	
	go func() {
		for {
			defer func() {
				if err := recover(); err != nil {
					PrintErr(err)
				}
			}()
			
			conn, err := lan.Accept()
			CheckErr(err)
			go handleLanConnection(conn, bc)
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
	
	printHubsBucket(bc.db)
	printTxsBucket(bc.db)
	
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
	
	go func() {
		for {
			defer func() {
				if err := recover(); err != nil {
					PrintErr(err)
				}
			}()
			
			time.Sleep(11 * time.Minute)
			fromBlockchaintoEach1(bc)
		}
	}()
	
	go func() {
		for {
			defer func() {
				if err := recover(); err != nil {
					PrintErr(err)
				}
			}()
			
			time.Sleep(13 * time.Minute)
			fromEach1toBlockchain(bc)
		}
	}()
	
	for {
		// StartMine(bc, 5000, 300000, 15)
		StartMine(bc, 15000, 900000, 10)
	}
	
	<-c
}

func printHubsBucket(db *bolt.DB) {
	fmt.Println("\nHubsBucket:\n")
	
	err := db.View(func(tx *bolt.Tx) error {
		hbs := tx.Bucket([]byte(hubsBucket))
		c := hbs.Cursor()
		
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			hub := string(k)
			fmt.Println(hub)
		}
		
		return nil
	})
	CheckErr(err)
}

func printTxsBucket(db *bolt.DB) {
	fmt.Println("\nTxsBucket:\n")
	
	err := db.View(func(tx *bolt.Tx) error {
		txsB := tx.Bucket([]byte(txsBucket))
		c := txsB.Cursor()
		
		for k, v := c.First(); k != nil; k, v = c.Next() {
			tx := DeserializeTransaction(v)
			
			fmt.Println(&tx)
			fmt.Println("\n")
			
			txSize := len(v)
			
			fmt.Println("tx size: ", txSize , "bytes\n")
		}
		
		return nil
	})
	CheckErr(err)
	
	fmt.Println("\n")
}

func handleLanConnection(conn net.Conn, bc *Blockchain) {
	defer func() {
		if err := recover(); err != nil {
			PrintErr(err)
		}
    }()
	
	request, err := ioutil.ReadAll(conn)
	CheckErr(err)
	command := bytesToCommand(request[:CommandLength])
	
	// fmt.Printf("%s Received %s command\n", time.Now().UTC(), command)

	switch command {
	case "nonce":
		handleNonce(request)
	default:
		// fmt.Println("Unknown command!")
	}

	conn.Close()
}

func handleNonce(request []byte) {
	var buff bytes.Buffer
	var payload nonze

	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	nonceFromMinersOfPool = payload.Nonce
}

func SendMiningJob(addr string, prepareData []byte, difficulty uint8) {
	payload := gobEncode(miningjob{nodeLanAddress, prepareData, difficulty})
	request := append(commandToBytes("miningjob"), payload...)
	sendLanData(addr, &request)
}

func SendConfirmedBlockInfo(addr string, bc *Blockchain) {
	var height uint32
	counter := 0
	
	bci := bc.Iterator()
	
	for {
		block := bci.Next()
		counter++
		
		if counter > int(spendableOutputConfirmations) {
			pubKeyHash := Base58Decode([]byte(MiningAddress))
			pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
			
			var array [20]byte
			copy(array[:], pubKeyHash)
			
			if block.Transactions[0].Vout[0].PubKeyHash == array {
				height = block.GetHeight()
				
				break
			}
		}
	}
	
	payload := gobEncode(confirmedblockinfo{nodeLanAddress, height})
	request := append(commandToBytes("confirmbcinfo"), payload...)
	sendLanData(addr, &request)
}

func sendLanData(addr string, data *[]byte) {
	conn, err := net.Dial(protocol, addr)
	
	if err != nil {
		msgBody := "系統警報：從 " + nodeLanAddress + " 傳送資料給 " + addr + " 時連線失敗。"
		SendAdminEmail(msgBody)
		
		return
	}
	
	defer conn.Close()
	
	_, err = io.Copy(conn, bytes.NewReader(*data))
	CheckErr(err)
}

func fromBlockchaintoEach1(bc *Blockchain) {
	db, err := sql.Open("mysql", each1.DB_EACH1)
	CheckErr(err)
	
	defer db.Close()
	
	BtoEConfirmations(db, bc)
}

func BtoEConfirmations(db *sql.DB, bc *Blockchain) {
	rows, err := db.Query("SELECT ID, UserID, CreateTime, Address, IsUseful FROM kd_dps WHERE Type=? AND IsPrint=? AND Deposit=?", blockChain, "1", "0")
	CheckErr(err)
	
	defer rows.Close()
	
	for rows.Next() {
		var id, userID, createTime, addr, isUseful string
		var amount uint
		isInBlockchain := false
		confirmations := 0
		threshold := 120
		
		err := rows.Scan(&id, &userID, &createTime, &addr, &isUseful)
		CheckErr(err)
		
		pubK := Base58Decode([]byte(addr))
		pubK = pubK[1 : len(pubK)-4]
		
		var pubKeyHash [20]byte
		copy(pubKeyHash[:], pubK)
		
		bci := bc.Iterator()
		
		for i := 0; i < 999; i++ {
			block := bci.Next()
			
			for index, tx := range block.Transactions {
				if index != 0 {					
					if tx.Vout[0].PubKeyHash == pubKeyHash {
						isInBlockchain = true
						amount = tx.Vout[0].Value
						
						if i < threshold {
							confirmations = i
							
						} else {
							confirmations = threshold
						}
					}
				}
				
				if isInBlockchain {
					break
				}
			}
			
			if isInBlockchain {
				break
			}
		}
		
		if isInBlockchain {
			if confirmations < threshold {
				stmt, err := db.Prepare("UPDATE kd_dps SET Amount=?, Confirmations=?, IsUseful=? WHERE ID=? LIMIT 1")
				CheckErr(err)
				_, err = stmt.Exec(amount, confirmations, "1", id)
				CheckErr(err)
				
			} else {
				coinBalance := each1.CoinBalance(userID, db)
				newBalance := coinBalance + amount
				
				tradeTime := time.Now().UTC()
				
				tx, err := db.Begin()
				CheckErr(err)
				
				stmt, err := tx.Prepare("UPDATE kd_dps SET Amount=?, Confirmations=?, IsPrint=?, IsUseful=?, Deposit=?, DepositTime=? WHERE ID=? LIMIT 1")
				CheckErr(err)
				_, err = stmt.Exec(amount, confirmations, "0", "1", "1", tradeTime, id)
				CheckErr(err)
				
				stmt, err = tx.Prepare("INSERT kd_dbf SET UserID=?, TradeTime=?, Type=?, Amount=?, Balance=?, Memo=?")
				CheckErr(err)
				_, err = stmt.Exec(userID, tradeTime, "D", amount, newBalance, "D8f")
				CheckErr(err)
				
				tx.Commit()
				
				title := strconv.Itoa(int(amount)) + " wakacoins was deposited into your Each1 account."
				url := "/wakacoin/passbook"
				
				each1.Notification(db, userID, title, url)
			}
			
		} else {
			t, err := time.Parse("2006-01-02 15:04:05", createTime)
			CheckErr(err)
			
			twelveDays := t.Add(+288 * time.Hour).Unix()
			now := time.Now().UTC().Unix()
			
			if now > twelveDays {
				if isUseful == "0" {
					stmt, err := db.Prepare("UPDATE kd_dps SET IsPrint=? WHERE ID=? LIMIT 1")
					CheckErr(err)
					_, err = stmt.Exec("0", id)
					CheckErr(err)
					
				} else {
					str := "ID." + id + " 從 " + blockChain + " blockchain 轉帳到 Each1.net 發生異常"
					PrintMessage(str)
					SendAdminEmail(str)
				}
			}
		}
	}
}

func fromEach1toBlockchain(bc *Blockchain) {
	db, err := sql.Open("mysql", each1.DB_EACH1)
	CheckErr(err)
	
	defer db.Close()
	
	EtoBCreate(db, bc)
	EtoBCreatingCheck(db)
	EtoBConfirmations(db, bc)
}

func EtoBCreate(db *sql.DB, bc *Blockchain) {
	rows, err := db.Query("SELECT ID, Address, Amount FROM kd_wtd WHERE Type=? AND isCreateTX=? AND isCreating=? AND Complete=?", blockChain, "0", "0", "0")
	CheckErr(err)
	
	defer rows.Close()
	
	for rows.Next() {
		var id, addr string
		var amount uint
		
		err := rows.Scan(&id, &addr, &amount)
		CheckErr(err)
		
		stmt, err := db.Prepare("UPDATE kd_wtd SET isCreating=?, lanchTime=? WHERE ID=? LIMIT 1")
		CheckErr(err)
		_, err = stmt.Exec("1", time.Now().UTC(), id)
		CheckErr(err)
		
		var tx *Transaction
		
		for {
			tx, err = NewUTXOTransaction("", addr, amount, bc)
			
			if err != nil {
				PrintErr(err)
				
				str := "從 Each1.net 轉帳到 " + blockChain + " blockchain 發生異常 - " + err.Error()
				SendAdminEmail(str)
				
				var errors uint8
				
				err := db.QueryRow("SELECT ErrorsA FROM kd_wtd WHERE ID=? LIMIT 1", id).Scan(&errors)
				CheckErr(err)
				
				errors++
				
				stmt, err := db.Prepare("UPDATE kd_wtd SET ErrorsA=? WHERE ID=? LIMIT 1")
				CheckErr(err)
				_, err = stmt.Exec(errors, id)
				CheckErr(err)
				
				return
			}
			
			isValid := tx.VerifySign(bc)
			
			if isValid {
				break
			}
		}
		
		err = tx.AddNewTxToBucket(bc.db)
		
		if err != nil {
			PrintErr(err)
			
			str := "從 Each1.net 轉帳到 " + blockChain + " blockchain 發生異常 - " + err.Error()
			SendAdminEmail(str)
			
			stmt, err := db.Prepare("UPDATE kd_wtd SET isCreating=? WHERE ID=? LIMIT 1")
			CheckErr(err)
			
			_, err = stmt.Exec("0", id)
			CheckErr(err)
			
		} else {
			txSerialize := tx.Serialize()
			
			txHex := fmt.Sprintf("%x", txSerialize)
			
			stmt, err := db.Prepare("UPDATE kd_wtd SET isCreateTX=?, TX=? WHERE ID=? LIMIT 1")
			CheckErr(err)
			
			_, err = stmt.Exec("1", txHex, id)
			CheckErr(err)
			
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
					sendTx(node, txSerialize, bc.db, false)
				}
			}
		}
	}
}

func EtoBCreatingCheck(db *sql.DB) {
	rows, err := db.Query("SELECT ID FROM kd_wtd WHERE Type=? AND isCreateTX=? AND isCreating=? AND lanchTime<?", blockChain, "0", "1", time.Now().UTC().Add(-30 * time.Minute))
	CheckErr(err)
	
	defer rows.Close()
	
	for rows.Next() {
		var id string
		
		err := rows.Scan(&id)
		CheckErr(err)
		
		var errors uint8
		
		err = db.QueryRow("SELECT ErrorsA FROM kd_wtd WHERE ID=? LIMIT 1", id).Scan(&errors)
		CheckErr(err)
		
		errors++
		
		stmt, err := db.Prepare("UPDATE kd_wtd SET ErrorsA=? WHERE ID=? LIMIT 1")
		CheckErr(err)
		_, err = stmt.Exec(errors, id)
		CheckErr(err)
		
		str := "從 Each1.net 轉帳到 " + blockChain + " blockchain 發生異常"
		PrintMessage(str)
		SendAdminEmail(str)
	}
}

func EtoBConfirmations(db *sql.DB, bc *Blockchain) {
	rows, err := db.Query("SELECT ID, ErrorsB, TX FROM kd_wtd WHERE Type=? AND isCreateTX=? AND Complete=?", blockChain, "1", "0")
	CheckErr(err)
	
	defer rows.Close()
	
	for rows.Next() {
		var id, txHex string
		var errorsB, confirmations uint8
		var isInBlockchain, isInTxsBucket bool
		
		err := rows.Scan(&id, &errorsB, &txHex)
		CheckErr(err)
		
		txSerialize, err := hex.DecodeString(txHex)
		CheckErr(err)
		
		tnx := DeserializeTransaction(txSerialize)
		
		bci := bc.Iterator()
		
		for i := 0; i < 999; i++ {
			block := bci.Next()
			
			for index, v := range block.Transactions {
				if index != 0 {
					if v.ID == tnx.ID {
						if v.Vin[0].Block == tnx.Vin[0].Block && v.Vin[0].Txid == tnx.Vin[0].Txid && v.Vin[0].Index == tnx.Vin[0].Index && v.Vout[0].Value == tnx.Vout[0].Value && v.Vout[0].PubKeyHash == tnx.Vout[0].PubKeyHash {
							isInBlockchain = true
							
							if i < int(spendableOutputConfirmations) {
								confirmations = uint8(i)
							} else {
								confirmations = spendableOutputConfirmations
							}
							
							break
						}
					}
				}
				
				if isInBlockchain {
					break
				}
			}
			
			if isInBlockchain {
				break
			}
		}
		
		if !isInBlockchain {
			err := bc.db.View(func(tx *bolt.Tx) error {
				txsB := tx.Bucket([]byte(txsBucket))
				t := txsB.Get(tnx.ID[:])
				
				if t != nil {
					v := DeserializeTransaction(t)
					
					if v.ID == tnx.ID {
						if v.Vin[0].Block == tnx.Vin[0].Block && v.Vin[0].Txid == tnx.Vin[0].Txid && v.Vin[0].Index == tnx.Vin[0].Index && v.Vout[0].Value == tnx.Vout[0].Value && v.Vout[0].PubKeyHash == tnx.Vout[0].PubKeyHash {
							isInTxsBucket = true
						}
					}
				}
				
				return nil
			})
			CheckErr(err)
		}
		
		if !isInBlockchain && !isInTxsBucket {
			errorsB++
			
			stmt, err := db.Prepare("UPDATE kd_wtd SET ErrorsB=? WHERE ID=? LIMIT 1")
			CheckErr(err)
			
			_, err = stmt.Exec(errorsB, id)
			CheckErr(err)
			
			str := "從 Each1.net 轉帳到 " + blockChain + " blockchain 發生異常"
			PrintMessage(str)
			SendAdminEmail(str)
		}
		
		if isInBlockchain {
			if confirmations == spendableOutputConfirmations {
				stmt, err := db.Prepare("UPDATE kd_wtd SET Confirmations=?, Complete=?, CompleteTime=? WHERE ID=? LIMIT 1")
				CheckErr(err)
				
				_, err = stmt.Exec(confirmations, "1", time.Now().UTC(), id)
				CheckErr(err)
				
			} else {
				stmt, err := db.Prepare("UPDATE kd_wtd SET Confirmations=? WHERE ID=? LIMIT 1")
				CheckErr(err)
				
				_, err = stmt.Exec(confirmations, id)
				CheckErr(err)
			}
		}
	}
}