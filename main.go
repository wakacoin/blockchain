package main

import (
	"fmt"
	"os"
	
	. "wakacoin"
)

func main() {
	os.Setenv("NODE_ID", "1024")
	
	if SetLocalhostStaticIPAddr {
		if len(LocalhostStaticIPAddr) == 0 {
			errMSG := "ERROR: Invalid LocalhostStaticIPAddr"
			
			fmt.Println("\n", errMSG)
			os.Exit(1)
		}
		
		err := ValidateAddrHost(LocalhostStaticIPAddr)
		
		if err != nil {
			fmt.Println("\n", err)
			os.Exit(1)
		}
	}
	
	if SetLocalhostDomainName {
		if len(LocalhostDomainName) == 0 {
			errMSG := "ERROR: Invalid LocalhostDomainName"
			
			fmt.Println("\n", errMSG)
			os.Exit(1)
		}
	}
	
	nodeID := os.Getenv("NODE_ID")
	fmt.Println("port: ", nodeID)
	
	cli := CLI{}
	cli.Run()
}