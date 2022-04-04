package wakacoin

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	
	"golang.org/x/crypto/ripemd160"
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func GetAddress(pubKeyHash [20]byte) []byte {
	versionedPayload := append([]byte{walletVersion}, pubKeyHash[:]...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
}

func GetPrivateKey(pubKeyHash [20]byte, wallets *Wallets) ecdsa.PrivateKey {
	address := fmt.Sprintf("%s", GetAddress(pubKeyHash))
	wallet := wallets.GetWallet(address)
	
	return wallet.PrivateKey
}

func NewWallet() *Wallet {
	private, public := newKeyPair()
	wallet := Wallet{private, public}

	return &wallet
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	CheckErr(err)
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}

func HashPubKey(pubKey []byte) [20]byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	CheckErr(err)
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)
	
	var array [20]byte
	copy(array[:], publicRIPEMD160)
	return array
}

func ValidateAddress(address string) bool {
	if len(address) != 34 {
		return false
	}

	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-int(addressChecksumLen):]
	walletVersion := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-int(addressChecksumLen)]
	targetChecksum := checksum(append([]byte{walletVersion}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}
