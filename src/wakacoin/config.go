package wakacoin

import (
	"math"
)

const(
	blockChain = "W"
	
	dbFile = "Wakacoin_%s.db"
	contractDbFile = "ContractChain_%s.db"
	
	paramBucket = "parameters"
	blocksBucket = "blocks"
	verifyBucket = "verifyBucket"
	utxoBucket_A = "utxoBucket_A"
	utxoBucket_B = "utxoBucket_B"
	contractBucket_A = "contractBucket_A"
	contractBucket_B = "contractBucket_B"
	hubsBucket = "hubs"
	txsBucket = "txs"
	
	walletFile = "wallet_%s.dat"
	
	blockVersion uint32 = 0
	maxBlockSize uint32 = 5242880   //5 MB
	stretch float32 = 1.01
	maxBlockHeaderPayload uint8 = 85
	coinbaseReservedSize uint16 = 700
	minTxSize uint8 = 150
	maxVins uint16 = 1001
	maxTxAmount uint16 = 5000
	
	genesisSubsidy uint32 = 5000
	halving uint32 = 2160
	maxHalvingRound uint32 = 13
	
	difficultyDefault_0 uint8 = 20
	difficultyDefault_1 uint8 = 20
	averageBlockTime uint16 = 600
	averageBlockTimeBlocks uint8 = 255
	averageBlockTimeBlocks_2 uint8 = 60
	averageBlockTimeBlocks_3 uint8 = 15
	tolerance uint16 = 300
	tolerance_2 uint16 = 60
	tolerance_3 uint16 = 10
	toleranceUpperLimit uint16 = 10
	toleranceLowerLimit uint16 = 200
	
	transactionVersion uint32 = 0
	spendableOutputConfirmations uint8 = 255
	confirmationsError = "The input has not yet reached 255 confirmations"
	
	utxoHaveToBeReindexed = "UTXO have to be reindexed"
	utxoCanNotBeUpdated = "UTXO can not be updated"
	
	walletVersion = byte(0x00)
	addressChecksumLen uint8 = 4
	
	upgradeNotice = "Please upgrade the software version"
	nodeVersion uint32 = 0
	protocol = "tcp"
	CommandLength uint8 = 14
	hubsCounter uint8 = 180
	hubsMax uint8 = 5
	txidsPacketMax uint8 = 3
	
	protectFromGenesis = "000004c8660c7e4bc9afcc6c0c51ad506ebda759d80fd9d0bd7a9538805b0da2"
	protectTo = "00000fcc1d759254c32ffdc3f4300e6b7d43afefd5d6d9461d811b90d1c350df"
	protectHeight uint32 = 99230

	contractVersion uint32 = 0
)

var (
	blockHashesFromGenesisToTip [][32]byte
	isRebuilding bool
	
	isVerifyBlocks bool
	isVerifying bool
	isUpdating bool
	inUse []string
	
	knownNodes = make(map[string]uint8)
	blacklist = make(map[string]int64)
	nodeId string
	nodeAddress string
	hostStaticAddress string
	hostDomainName string
	localhostisDefaultHub bool
	MiningAddress string
	
	WebServerLanAddress string
	nodeLanAddress string
	miningPool bool
	nonceFromMinersOfPool uint64
	
	b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
)

func Subsidy(blockHeight uint32) uint32 {
	switch {
	case blockHeight < halving:
		return genesisSubsidy
		
	case blockHeight < halving*maxHalvingRound:
		h := float64(halving)
		n := float64(blockHeight)
		x := n / h
		xInt := int(math.Floor(x))

		s := float64(genesisSubsidy)
		for i := 0; i < xInt; i++ {
			s = s / 2
		}
		return uint32(math.Ceil(s))
		
	default:
		return 1
	}
}
