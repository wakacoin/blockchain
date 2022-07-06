package wakacoin

import (
	"bytes"
	"encoding/gob"
	"math/rand"
	"net"
	"time"
	
	"github.com/boltdb/bolt"
)

type gethubs struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
}

type hubz struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
	Hubs        []string
}

type getknownnodes struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
}

type nodespacket struct {
	BlockChain  string
	NodeVersion uint32
	StaticAddr  bool
	Host        string
	Port        string
	NodesPacket []string
}

func ConnectDefaultHub(db *bolt.DB) {
	if !localhostisDefaultHub {
		sendGetHubs(DefaultHub, db, false)
	}
}

func ConnectHubs(db *bolt.DB) {
	var hubsBucketNeedRebuild bool
	var hubs []string
	
	err := db.View(func(tx *bolt.Tx) error {
		hbs := tx.Bucket([]byte(hubsBucket))
		c := hbs.Cursor()
		
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			hub := string(k)
			
			addrHost, _, err := net.SplitHostPort(hub)
			CheckErr(err)
			
			if err := ValidateAddrHost(addrHost); err != nil {
				hubsBucketNeedRebuild = true
				break
			}
			
			allowAppend := true
			
			switch {
			case hub == nodeAddress:
				allowAppend = false
				
			case SetLocalhostDomainName && hub == hostDomainName:
				allowAppend = false
				
			case SetLocalhostStaticIPAddr && hub == hostStaticAddress:
				allowAppend = false
			}
			
			if allowAppend {
				hubs = append(hubs, hub)
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	if hubsBucketNeedRebuild {
		err := db.Update(func(tx *bolt.Tx) error {
			bucketName := []byte(hubsBucket)
			
			err := tx.DeleteBucket(bucketName)
			
			if err != nil && err != bolt.ErrBucketNotFound {
				CheckErr(err)
			}
			
			_, err = tx.CreateBucket(bucketName)
			CheckErr(err)
			
			return nil
		})
		CheckErr(err)
	}
	
	for _, hub := range hubs {
		sendGetKnownNodes(hub, db, false)
	}
}

func handleGetHubs(remoteAddrHost string, request []byte, db *bolt.DB) {
	var buff bytes.Buffer
	var payload gethubs
	
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
	
	hubs := []string{}
	
	err = db.View(func(tx *bolt.Tx) error {
		h := tx.Bucket([]byte(hubsBucket))
		c := h.Cursor()
		
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			hub := string(k)
			
			if len(hubs) < int(hubsMax) {
				addrHost, _, err := net.SplitHostPort(hub)
				CheckErr(err)
				
				if err := ValidateAddrHost(addrHost); err == nil {
					if hub != remoteAddr {
						hubs = append(hubs, hub)
					}
				}
				
			} else {
				break
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	if len(hubs) == 0 {
		hubs = append(hubs, DefaultHub)
	}
	
	sendHubs(remoteAddr, db, &hubs, false)
}

func handleHubs(remoteAddrHost string, request []byte, db *bolt.DB) {
	var buff bytes.Buffer
	var payload hubz
	
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
	
	hubs := []string{}
	
	err = db.Update(func(tx *bolt.Tx) error {
		hbs := tx.Bucket([]byte(hubsBucket))
		
		for _, v := range payload.Hubs {
			if len(hubs) < int(hubsMax) {
				addrHost, _, err := net.SplitHostPort(v)
				CheckErr(err)
				
				if err := ValidateAddrHost(addrHost); err != nil {
					break
				}
				
				err = hbs.Put([]byte(v), nil)
				CheckErr(err)
				
				var hubExists bool
				
				for _, hub := range hubs {
					if hub == v {
						hubExists = true
					}
				}
				
				if !hubExists {
					hubs = append(hubs, v)
				}
				
			} else {
				break
			}
		}
		
		return nil
	})
	CheckErr(err)
	
	for _, hub := range hubs {
		sendGetKnownNodes(hub, db, false)
	}
}

func handleGetKnownNodes(remoteAddrHost string, request []byte, db *bolt.DB) {
	var buff bytes.Buffer
	var payload getknownnodes
	
	buff.Write(request[CommandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	CheckErr(err)
	
	err = ValidateBlockChain(payload.BlockChain)
	CheckErr(err)
	
	remoteAddr, err := RemoteAddr(payload.NodeVersion ,payload.StaticAddr, payload.Host, payload.Port, remoteAddrHost)
	CheckErr(err)
	
	nodeExists, counter := nodeIsKnown(remoteAddr)
	manageKnownNodes(nodeExists, remoteAddr)
	
	if nodeExists {
		if counter < hubsCounter {
			if counter < 255 {
				counter++
				knownNodes.Store(remoteAddr, counter)
			}
			
		} else {
			err := db.Update(func(tx *bolt.Tx) error {
				hCounter := 0
				hubExists := false
				
				h := tx.Bucket([]byte(hubsBucket))
				c := h.Cursor()
				
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					hub := string(k)
					
					if hub == remoteAddr {
						hubExists = true
					}
					
					hCounter++
				}
				
				if !hubExists {
					if hCounter < int(hubsMax) {
						err := h.Put([]byte(remoteAddr), nil)
						CheckErr(err)
						
					} else {
						err := h.Delete([]byte(nodeAddress))
						CheckErr(err)
						
						err = h.Delete([]byte(DefaultHub))
						CheckErr(err)
						
						if SetLocalhostDomainName {
							err := h.Delete([]byte(hostDomainName))
							CheckErr(err)
						}
						
						if SetLocalhostStaticIPAddr {
							err := h.Delete([]byte(hostStaticAddress))
							CheckErr(err)
						}
						
						i := 0
						
						for k, _ := c.First(); k != nil; k, _ = c.Next() {
							i++
						}
						
						if i < int(hubsMax) {
							ifPutNode := true
							
							switch {
							case remoteAddr == nodeAddress:
								ifPutNode = false
							
							case remoteAddr == DefaultHub:
								ifPutNode = false
								
							case SetLocalhostDomainName && remoteAddr == hostDomainName:
								ifPutNode = false
								
							case SetLocalhostStaticIPAddr && remoteAddr == hostStaticAddress:
								ifPutNode = false
							}
							
							if ifPutNode {
								err := h.Put([]byte(remoteAddr), nil)
								CheckErr(err)
							}
						}
					}
				}
				
				return nil
			})
			CheckErr(err)
		}
	}
	
	nodesPacket := packNodesPacket(10)

	if len(*nodesPacket) == 0 {
		*nodesPacket = append(*nodesPacket, DefaultHub)
	}

	nodesPacketRemoveRemoteAddr := []string{}

	for _, node := range *nodesPacket {
		if node != remoteAddr {
			nodesPacketRemoveRemoteAddr = append(nodesPacketRemoveRemoteAddr, node)
		}
	}
	
	if len(nodesPacketRemoveRemoteAddr) > 0 {
		sendKnownNodesPacket(remoteAddr, db, &nodesPacketRemoveRemoteAddr, false)
	}
}

func handleKnownNodesPacket(remoteAddrHost string, request []byte) {
	var buff bytes.Buffer
	var payload nodespacket
	
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
	
	for _, v := range payload.NodesPacket {
		nodeExists, _ := nodeIsKnown(v)
		manageKnownNodes(nodeExists, v)
	}
}

func sendGetHubs(addr string, db *bolt.DB, report bool) {
	data := gethubs{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId}
	payload := gobEncode(data)
	request := append(commandToBytes("gethubs"), payload...)
	sendData(addr, &request, db, report)
}

func sendHubs(addr string, db *bolt.DB, hubList *[]string, report bool) {
	data := hubz{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId, *hubList}
	payload := gobEncode(data)
	request := append(commandToBytes("hubs"), payload...)
	sendData(addr, &request, db, report)
}

func sendGetKnownNodes(addr string, db *bolt.DB, report bool) {
	data := getknownnodes{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId}
	payload := gobEncode(data)
	request := append(commandToBytes("getknownnodes"), payload...)
	sendData(addr, &request, db, report)
}

func sendKnownNodesPacket(addr string, db *bolt.DB, nodesPacket *[]string, report bool) {
	data := nodespacket{blockChain, nodeVersion, SetLocalhostStaticIPAddr, LocalhostStaticIPAddr, nodeId, *nodesPacket}
	payload := gobEncode(data)
	request := append(commandToBytes("knownodepacket"), payload...)
	sendData(addr, &request, db, report)
}

func packNodesPacket(nodesPacketMax uint8) *[]string {
	nodesPacket := []string{}
	var counter uint8 = 0
	
	if nodesPacketMax < 1 {
		nodesPacketMax = 1
	}

	knownNodes.Range(func(k, v interface{}) bool {
		rand.Seed(time.Now().UnixNano())

		if counter < nodesPacketMax {
			if rand.Intn(30) % 2 == 1 {
				nodesPacket = append(nodesPacket, k.(string))
			}
		}

		if counter < 255 {
			counter++
		}
		
		return true
	})

	if len(nodesPacket) == 0 {
		if counter > 0 {
			counter = 0

			knownNodes.Range(func(k, v interface{}) bool {
				if counter < nodesPacketMax {
					nodesPacket = append(nodesPacket, k.(string))
				}

				if counter < 255 {
					counter++
				}
				
				return true
			})
		}
	}

	return &nodesPacket
}
