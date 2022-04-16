package wakacoin

import (
	"os"
)

func (cli *CLI) send(from, to string, amount uint, nodeID string) {
	if amount < 1 || amount > 12345678 {
		str := "ERROR: Amount is not valid."
		PrintMessage(str)
		os.Exit(1)
	}
	
	if ValidateAddress(to) != true {
		str := "ERROR: The recipient’s address is not valid."
		PrintMessage(str)
		os.Exit(1)
	}
	
	if len(from) > 0 {
		if ValidateAddress(from) != true {
			str := "ERROR: The sender’s address is not valid."
			PrintMessage(str)
			os.Exit(1)
		}
		
		if from == to {
			str := "ERROR: The sender’s address and the recipient’s address cannot be the same."
			PrintMessage(str)
			os.Exit(1)
		}
	}
	
	str := "Update blockchain, please wait 10 minutes."
	PrintMessage(str)
	
	var minerAddress string
	sendNewTx := true
	
	StartServer(nodeID, minerAddress, from, to, sendNewTx, uint32(amount))
}
