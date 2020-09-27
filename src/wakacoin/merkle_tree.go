package wakacoin

import (
	"crypto/sha256"
)

func NewMerkleTree(data [][32]byte) [32]byte {
	for {
		if len(data)%2 != 0 {
			data = append(data, data[len(data)-1])
		}
		
		var nodes [][32]byte
		
		x := len(data)
		
		for i := 0; i < x; i += 2 {
			node := NewMerkleNode(data[i], data[i+1])
			nodes = append(nodes, node)
		}
		
		if len(nodes) == 1 {
			return nodes[0]
		}
		
		data = nil
		data = append(data, nodes...)
	}
}

func NewMerkleNode(left, right [32]byte) [32]byte {
	hashes := append(left[:], right[:]...)
	hash := sha256.Sum256(hashes)
	
	return hash
}
