package wakacoin

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type Wallets struct {
	Wallets map[string]*Wallet
}

func NewWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile(nodeID)

	return &wallets, err
}

func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	pubKeyHash := HashPubKey(wallet.PublicKey)
	address := fmt.Sprintf("%s", GetAddress(pubKeyHash))

	ws.Wallets[address] = wallet

	return address
}

func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address, _ := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) GetWallet(address string) (wallet *Wallet, err error) {
	addresses := ws.GetAddresses()

	for _, addr := range addresses {
		if addr == address {
			return ws.Wallets[address], nil
		}
	}

	return wallet, errors.New("ERROR: You do not have the private key of the address.")
}

func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFileName := fmt.Sprintf(walletFile, nodeID)
	
	if _, err := os.Stat(walletFileName); err != nil {
		if os.IsNotExist(err) {
			return errors.New("ERROR: The wallet file does not exist.")
		} else {
			return errors.New("ERROR: There is an error when reading the wallet file.")
		}
	}

	fileContent, err := ioutil.ReadFile(walletFileName)
	CheckErr(err)

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	CheckErr(err)

	ws.Wallets = wallets.Wallets

	return nil
}

func (ws Wallets) SaveToFile(nodeID string) {
	var content bytes.Buffer
	walletFileName := fmt.Sprintf(walletFile, nodeID)

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	CheckErr(err)

	err = ioutil.WriteFile(walletFileName, content.Bytes(), 0644)
	CheckErr(err)
}
