package wakacoin

import (
	"bytes"
	"encoding/gob"
)

type TXOutput struct {
	Value      uint32
	PubKeyHash [20]byte
}

func NewTXOutput(value uint32, address []byte) TXOutput {
	txo := TXOutput{value, [20]byte{}}
	txo.Lock(address)

	return txo
}

func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	
	var array [20]byte
	copy(array[:], pubKeyHash)
	out.PubKeyHash = array
}

type UTXOutput struct {
	Index      int8
	Value      uint32
	PubKeyHash [20]byte
	Height     uint32
}

type UTXOutputs struct {
	Outputs []*UTXOutput
}

func (outs *UTXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	CheckErr(err)

	return buff.Bytes()
}

func DeserializeUTXOutputs(data []byte) UTXOutputs {
	var outputs UTXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	CheckErr(err)

	return outputs
}
