package wakacoin

type TXInput struct {
	Block     [32]byte
	Txid      [32]byte
	Index     int8
	Signature []byte
	PubKey    []byte
}
