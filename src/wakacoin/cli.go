package wakacoin

import (
	"flag"
	"fmt"
	"os"
)

type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  contractchaincreate -address ADDRESS -filename FILENAME - Create a contract chain and send genesis block hash to blockchain")
	fmt.Println("  contractchainvalidate -address ADDRESS - Validate a contract chain")
	fmt.Println("  contractcreate -address ADDRESS -filename FILENAME - Create a contract and send the block hash to blockchain")
	fmt.Println("  contractlistfiles -address ADDRESS - List all the name of files in the contract chain")
	fmt.Println("  contractreleasefile -address ADDRESS -height HEIGHT - Release the file in the contract chain by HEIGHT")
	// fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  createwallet - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  datacleansing - Delete data that is completely useless")
	fmt.Println("  getbalance -address ADDRESS -detail BOOL - Get balance of ADDRESS. -detail print details (false/true)")
	fmt.Println("  listaddresses - List all addresses from the wallet file")
	fmt.Println("  printchain -page PAGE - Print all the blocks of the blockchain by PAGE")
	fmt.Println("  printransaction -blockid BLOCKID -txid TXID - Print the transaction of the block")
	fmt.Println("  printxsbucket - Print all the transactions in the txsBucket")
	fmt.Println("  reindexutxo - Rebuilds the UTXO set")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send AMOUNT of coins from FROM address to TO address")
	fmt.Println("  sendtxs - Send all the transactions in the txsBucket")
	fmt.Println("  startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
	fmt.Println("  statistic - The statistic of the blockchain")
	// fmt.Println("  tool - Develop temporary functions")
	fmt.Println("  validate - Validate the blockchain")
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
	
	contractChainCreateCmd := flag.NewFlagSet("contractchaincreate", flag.ExitOnError)
	contractChainValidateCmd := flag.NewFlagSet("contractchainvalidate", flag.ExitOnError)
	contractCreateCmd := flag.NewFlagSet("contractcreate", flag.ExitOnError)
	contractListFilesCmd := flag.NewFlagSet("contractlistfiles", flag.ExitOnError)
	contractReleaseFileCmd := flag.NewFlagSet("contractreleasefile", flag.ExitOnError)
	// createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	dataCleansingCmd := flag.NewFlagSet("datacleansing", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	prinTransactionCmd := flag.NewFlagSet("printransaction", flag.ExitOnError)
	prinTxsBucketCmd := flag.NewFlagSet("printxsbucket", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendTXsCmd := flag.NewFlagSet("sendtxs", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)
	statisticCmd := flag.NewFlagSet("statistic", flag.ExitOnError)
	toolCmd := flag.NewFlagSet("tool", flag.ExitOnError)
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	
	contractChainCreateAddress := contractChainCreateCmd.String("address", "", "The address to manage the contract chain")
	contractChainCreateFilename := contractChainCreateCmd.String("filename", "", "The file to be placed in the contract")
	contractChainValidateAddress := contractChainValidateCmd.String("address", "", "The address of the contract chain")
	contractCreateAddress := contractCreateCmd.String("address", "", "The address to manage the contract chain")
	contractCreateFilename := contractCreateCmd.String("filename", "", "The file to be placed in the contract")
	contractListFilesAddress := contractListFilesCmd.String("address", "", "The address of the contract chain")
	contractReleaseFileAddress := contractReleaseFileCmd.String("address", "", "The address of the contract chain")
	contractReleaseFileHeight := contractReleaseFileCmd.Uint("height", 0, "Height of block")
	// createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	getBalanceDetail := getBalanceCmd.Bool("detail", false, "whether to print out the details")
	printchainPage := printChainCmd.Uint("page", 0, "Number of page")
	printransactionBlockid := prinTransactionCmd.String("blockid", "", "Block header hash")
	printransactionTxid := prinTransactionCmd.String("txid", "", "Transaction ID")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Uint("amount", 0, "Amount to send")
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")
	
	switch os.Args[1] {
	case "contractchaincreate":
		err := contractChainCreateCmd.Parse(os.Args[2:])
		CheckErr(err)

	case "contractchainvalidate":
		err := contractChainValidateCmd.Parse(os.Args[2:])
		CheckErr(err)

	case "contractcreate":
		err := contractCreateCmd.Parse(os.Args[2:])
		CheckErr(err)

	case "contractlistfiles":
		err := contractListFilesCmd.Parse(os.Args[2:])
		CheckErr(err)

	case "contractreleasefile":
		err := contractReleaseFileCmd.Parse(os.Args[2:])
		CheckErr(err)

	/* case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		CheckErr(err)
	*/
	
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "datacleansing":
		err := dataCleansingCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		CheckErr(err)
	
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		CheckErr(err)
	
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "printransaction":
		err := prinTransactionCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "printxsbucket":
		err := prinTxsBucketCmd.Parse(os.Args[2:])
		CheckErr(err)
	
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "sendtxs":
		err := sendTXsCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "statistic":
		err := statisticCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "tool":
		err := toolCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	case "validate":
		err := validateCmd.Parse(os.Args[2:])
		CheckErr(err)
		
	default:
		cli.printUsage()
		os.Exit(1)
	}
	
	if contractChainCreateCmd.Parsed() {
		if *contractChainCreateAddress == "" || *contractChainCreateFilename == "" {
			contractChainCreateCmd.Usage()
			os.Exit(1)
		}
		
		cli.contractChainCreate(*contractChainCreateAddress, *contractChainCreateFilename, nodeID)
	}

	if contractChainValidateCmd.Parsed() {
		if *contractChainValidateAddress == "" {
			contractChainValidateCmd.Usage()
			os.Exit(1)
		}
		
		cli.contractChainValidate(*contractChainValidateAddress, nodeID)
	}

	if contractCreateCmd.Parsed() {
		if *contractCreateAddress == "" || *contractCreateFilename == "" {
			contractCreateCmd.Usage()
			os.Exit(1)
		}
		
		cli.contractCreate(*contractCreateAddress, *contractCreateFilename, nodeID)
	}

	if contractListFilesCmd.Parsed() {
		if *contractListFilesAddress == "" {
			contractListFilesCmd.Usage()
			os.Exit(1)
		}
		
		cli.contractListFiles(*contractListFilesAddress)
	}

	if contractReleaseFileCmd.Parsed() {
		if *contractReleaseFileAddress == "" {
			contractReleaseFileCmd.Usage()
			os.Exit(1)
		}
		
		cli.contractReleaseFile(*contractReleaseFileAddress, *contractReleaseFileHeight)
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
	
	if dataCleansingCmd.Parsed() {
		cli.dataCleansing(nodeID)
	}
	
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		
		cli.getBalance(*getBalanceAddress, nodeID, *getBalanceDetail)
	}
	
	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
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

	if prinTxsBucketCmd.Parsed() {
		cli.prinTxsBucket(nodeID)
	}
	
	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}
	
	if sendCmd.Parsed() {
		if *sendTo == "" || *sendAmount < 1 {
			sendCmd.Usage()
			os.Exit(1)
		}
		
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID)
	}
	
	if sendTXsCmd.Parsed() {
		cli.sendTXs(nodeID)
	}
	
	if startNodeCmd.Parsed() {
		var from, to string
		var sendNewTx bool
		var amount uint32
		
		cli.startNode(nodeID, *startNodeMiner, from, to, sendNewTx, amount)
	}
	
	if statisticCmd.Parsed() {
		cli.statistic(nodeID)
	}
	
	if toolCmd.Parsed() {
		cli.tool(nodeID)
	}
	
	if validateCmd.Parsed() {
		cli.validate(nodeID)
	}
}
