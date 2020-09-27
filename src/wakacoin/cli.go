package wakacoin

import (
	"flag"
	"fmt"
	"os"
)

type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	// fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  createwallet - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  listaddresses - Lists all addresses from the wallet file")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  printchain -page PAGE - Print all the blocks of the blockchain by PAGE")
	fmt.Println("  printransaction -blockid BLOCKID -txid TXID - Print the transaction of the block")
	fmt.Println("  validate - Validate the blockchain")
	fmt.Println("  datacleansing - Delete data that is completely useless")
	fmt.Println("  reindexutxo - Rebuilds the UTXO set")
	fmt.Println("  statistic - The statistic of the blockchain")
	fmt.Println("  tool - Develop temporary functions")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send AMOUNT of coins from FROM address to TO")
	fmt.Println("  printxsbucket - Print all the transactions in the txsBucket")
	fmt.Println("  sendtxs - Send all the transactions in the txsBucket")
	fmt.Println("  startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()
	
	nodeID := os.Getenv("NODE_ID")
	
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!")
		os.Exit(1)
	}
	
	// createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	prinTransactionCmd := flag.NewFlagSet("printransaction", flag.ExitOnError)
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	dataCleansingCmd := flag.NewFlagSet("datacleansing", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	statisticCmd := flag.NewFlagSet("statistic", flag.ExitOnError)
	toolCmd := flag.NewFlagSet("tool", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	prinTxsBucketCmd := flag.NewFlagSet("printxsbucket", flag.ExitOnError)
	sendTXsCmd := flag.NewFlagSet("sendtxs", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)
	
	// createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	printchainPage := printChainCmd.Uint("page", 0, "Number of page")
	printransactionBlockid := prinTransactionCmd.String("blockid", "", "Block header hash")
	printransactionTxid := prinTransactionCmd.String("txid", "", "Transaction ID")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Uint("amount", 0, "Amount to send")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")
	
	switch os.Args[1] {
	/* case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		CheckErr(err)
	*/
	
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		CheckErr(err)
	
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		CheckErr(err)
	
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "printransaction":
		err := prinTransactionCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "validate":
		err := validateCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "datacleansing":
		err := dataCleansingCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "statistic":
		err := statisticCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "tool":
		err := toolCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "printxsbucket":
		err := prinTxsBucketCmd.Parse(os.Args[2:])
		CheckErr(err)
	
	case "sendtxs":
		err := sendTXsCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	default:
		cli.printUsage()
		os.Exit(1)
	}
	
	/*
		if createBlockchainCmd.Parsed() {
			if *createBlockchainAddress == "" {
				createBlockchainCmd.Usage()
				os.Exit(1)
			}
			
			cli.createBlockchain(*createBlockchainAddress, nodeID)
		}
	*/
	
	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}
	
	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}
	
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		
		cli.getBalance(*getBalanceAddress, nodeID)
	}
	
	if printChainCmd.Parsed() {
		if *printchainPage <= 0 {
			printChainCmd.Usage()
			os.Exit(1)
		}
		
		cli.printChain(*printchainPage, nodeID)
	}
	
	if prinTransactionCmd.Parsed() {
		if *printransactionBlockid == "" || *printransactionTxid == "" {
			prinTransactionCmd.Usage()
			os.Exit(1)
		}
		
		cli.prinTransaction(*printransactionBlockid, *printransactionTxid, nodeID)
	}

	if validateCmd.Parsed() {
		cli.validate(nodeID)
	}
	
	if dataCleansingCmd.Parsed() {
		cli.dataCleansing(nodeID)
	}
	
	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}
	
	if statisticCmd.Parsed() {
		cli.statistic(nodeID)
	}
	
	if toolCmd.Parsed() {
		cli.tool(nodeID)
	}
	
	if sendCmd.Parsed() {
		if *sendTo == "" || *sendAmount < 1 {
			sendCmd.Usage()
			os.Exit(1)
		}
		
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID)
	}
	
	if prinTxsBucketCmd.Parsed() {
		cli.prinTxsBucket(nodeID)
	}
	
	if sendTXsCmd.Parsed() {
		cli.sendTXs(nodeID)
	}
	
	if startNodeCmd.Parsed() {
		var from, to string
		var sendNewTx bool
		var amount uint
		
		cli.startNode(nodeID, *startNodeMiner, from, to, sendNewTx, amount)
	}
}
