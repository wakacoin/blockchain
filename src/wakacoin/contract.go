package wakacoin

import (
	"encoding/hex"
	"errors"
)

func ContractChainBlockHashDecodeString(contractChainBlockHashString string) (contractChainBlockHash [20]byte, err error) {
	errMSG := "ERROR: The Contract Chain Block Hash String is not valid."

	if len(contractChainBlockHashString) != 40 {
		return contractChainBlockHash, errors.New(errMSG)
	}
	
	hashByte, err := hex.DecodeString(contractChainBlockHashString)

	if err != nil {
		return contractChainBlockHash, errors.New(errMSG)
	}
	
	copy(contractChainBlockHash[:], hashByte)
	return contractChainBlockHash, nil
}
