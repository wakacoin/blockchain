package wakacoin

import (
	"math"
)

const(
	blockChain = "W"
	
	dbFile = "Wakacoin_%s.db"
	
	paramBucket = "parameters"
	blocksBucket = "blocks"
	verifyBucket = "verifyBucket"
	utxoBucket_A = "utxoBucket_A"
	utxoBucket_B = "utxoBucket_B"
	hubsBucket = "hubs"
	txsBucket = "txs"
	
	walletFile = "wallet_%s.dat"
	
	blockVersion uint32 = 0
	maxBlockSize uint32 = 5*1024*1024
	stretch float32 = 1.01
	maxBlockHeaderPayload uint8 = 85
	coinbaseReservedSize uint16 = 700
	minTxSize uint8 = 150
	maxVins uint16 = 1001
	maxTxAmount uint16 = 5000
	
	genesisSubsidy uint = 5000
	halving uint32 = 2160
	maxHalvingRound uint32 = 13
	
	difficultyDefault_0 uint8 = 20
	difficultyDefault_1 uint8 = 20
	averageBlockTimeOfBlocks uint8 = 255
	averageBlockTime uint16 = 600
	errorTolerance uint16 = 300
	
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
	hubsMax uint8 = 3
	knownNodesMax uint8 = 6
	knownNodesPacketMax uint8 = 2
	txidsPacketMax uint8 = 3
	
	protectFromGenesis = "000004c8660c7e4bc9afcc6c0c51ad506ebda759d80fd9d0bd7a9538805b0da2"
	protectTo = "0000025753fb9696efa410c33bf1a44bc0d8732634e5a7c98c1220875d435be9"
	protectHeight uint32 = 23324
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

func Subsidy(blockHeight uint32) uint {
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
		return uint(math.Ceil(s))
		
	default:
		return 1
	}
}
