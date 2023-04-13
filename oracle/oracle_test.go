package oracle

import (
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// This file contains almost end to end tests, where the only mocked part is the
// data that is fetched onchain: blocks, subscriptions, unsubscriptions and donations

// TODO: Test merkle roots and proofs generation

// TODO:
func Test_Oracle_ManualSubscription(t *testing.T) {
	oracle := NewOracle(&config.Config{
		Network:               "",
		PoolAddress:           "0xdead000000000000000000000000000000000000",
		UpdaterAddress:        "",
		DeployedSlot:          uint64(50000),
		CheckPointSizeInSlots: uint64(100),
		PoolFeesPercent:       5,
		PoolFeesAddress:       "0xfee0000000000000000000000000000000000000",
		CollateralInWei:       big.NewInt(1000000),
	})

	// Manually subscribe 3 validators with enogh collateral
	subs := GenerateSubsctiptions(
		/*valIndexs*/ []uint64{400000, 500000, 700000},
		/*valKeys*/ []string{"0xval_400000", "0xval_500000", "0xval_700000"},
		/*collaterals*/ []*big.Int{big.NewInt(1000000), big.NewInt(1000000), big.NewInt(1000000)},
		/*blockNums*/ []uint64{500, 500, 500},
		/*txHashes*/ []string{"0x1", "0x2", "0x3"},
		/*depositAddrs*/ []string{"0xaaa0000000000000000000000000000000000000", "0xaaa0000000000000000000000000000000000000", "0xccc0000000000000000000000000000000000000"},
	)

	// Process block with 3 subscriptions (no reward sent to pool)
	processedSlot, err := oracle.AdvanceStateToNextSlot(WrongFeeBlock(50000, 1, "0x"), subs, []Unsubscription{}, []Donation{})
	require.NoError(t, err)

	// Advance the state with 10 block without proposals to the smoothing pool
	for i := 1; i <= 10; i++ {
		oracle.AdvanceStateToNextSlot(WrongFeeBlock(50000+uint64(i), 1, "0x"), []Subscription{}, []Unsubscription{}, []Donation{})
	}

	// Validator 40000 proposes a block
	block1 := Block{
		Slot: uint64(50011), ValidatorIndex: uint64(400000),
		ValidatorKey: "0xval_400000", BlockType: OkPoolProposal,
		Reward: big.NewInt(245579896737171752), RewardType: MevBlock, DepositAddress: "0xaaa0000000000000000000000000000000000000",
	}

	processedSlot, err = oracle.AdvanceStateToNextSlot(block1, []Subscription{}, []Unsubscription{}, []Donation{})
	require.NoError(t, err)
	require.Equal(t, uint64(50011), processedSlot)

	// Validator 500000 proposes a block
	block2 := Block{
		Slot: uint64(50012), ValidatorIndex: uint64(500000),
		ValidatorKey: "0xval_500000", BlockType: OkPoolProposal,
		Reward: big.NewInt(945579196337171700), RewardType: MevBlock, DepositAddress: "0xaaa0000000000000000000000000000000000000",
	}

	processedSlot, err = oracle.AdvanceStateToNextSlot(block2, []Subscription{}, []Unsubscription{}, []Donation{})
	require.NoError(t, err)
	require.Equal(t, uint64(50012), processedSlot)

	enough := oracle.State.StoreLatestOnchainState()
	require.True(t, enough)

	require.Equal(t, "df67cc0d6a1d8b80f7d73b42813952c0e4d3936f597959fe87374eb89f100f5e", oracle.State.LatestCommitedState.MerkleRoot)

	// What we owe
	totalLiabilities := big.NewInt(0)
	for _, val := range oracle.State.Validators {
		totalLiabilities.Add(totalLiabilities, val.AccumulatedRewardsWei)
		totalLiabilities.Add(totalLiabilities, val.PendingRewardsWei)
	}
	totalLiabilities.Add(totalLiabilities, oracle.State.PoolAccumulatedFees) // TODO: rename wei

	// What we have (block fees + collateral)
	totalAssets := big.NewInt(0)
	totalAssets.Add(totalAssets, big.NewInt(245579896737171752)) // reward first block
	totalAssets.Add(totalAssets, big.NewInt(945579196337171700)) // reward second block
	for _, val := range oracle.State.Validators {
		totalAssets.Add(totalAssets, val.CollateralWei)
	}

	require.Equal(t, totalAssets, totalLiabilities)
}

// TODO: Mix manual and automatic subscriptions

// Simulates 100 slots with "AdvanceStateToNextSlot". Each slot is configured randomly with a
// new sub, unsub or donation. The block proposed can be okproposal, missed or wrongfee.
// these are all randomly set each block

func Test_100_slots_test(t *testing.T) {
	numBlocks := 100
	log.Infof("Number of blocks to simulate: %d", numBlocks)
	//set new oracle instance
	oracle := NewOracle(&config.Config{
		Network:               "mainnet",
		PoolAddress:           "0xdead000000000000000000000000000000000000",
		UpdaterAddress:        "",
		DeployedSlot:          uint64(50000),
		CheckPointSizeInSlots: uint64(100),
		PoolFeesPercent:       5,
		PoolFeesAddress:       "0xfee0000000000000000000000000000000000000",
		CollateralInWei:       big.NewInt(1000000),
	})

	subsIndex := make([]uint64, 0)
	totalAssets := big.NewInt(0)
	const seed int64 = 50000
	rand.Seed(seed)
	log.WithFields(log.Fields{
		"Execution seed": seed,
	})
	// main loop, iterates through 100 slots
	for i := 0; i <= numBlocks; i++ {
		newSubscription := make([]Subscription, 0)
		newUnsubscription := make([]Unsubscription, 0)
		don := make([]Donation, 0)
		fmt.Println("")
		log.Infoln("NEW BLOCK:")
		//throw dice to determine if a new subscription is set in this slot. 1/2 chance
		dice := rand.Intn(2)
		if dice == 0 {
			newSubscription = GenerateSubsctiptions(
				/*valIndexs*/ []uint64{50000 + uint64(i)},
				/*valKeys*/ []string{"val" + strconv.FormatUint(50000+uint64(i), 10)},
				/*collaterals*/ []*big.Int{big.NewInt(1000000)},
				/*blockNums*/ []uint64{50000 + uint64(i)},
				/*txHashes*/ []string{"0x1"},
				/*depositAddrs*/ []string{"0xaaa0000000000000000000000000000000000000"},
			)
			subsIndex = append(subsIndex, newSubscription[0].ValidatorIndex)
			totalAssets.Add(totalAssets, newSubscription[0].Collateral)
		}

		//throw dice to determine if a new unsubscription is set in this slot. 1/3 chance
		//(can only unsubscribe already subbed validators)
		dice = rand.Intn(3)
		if dice == 0 && len(subsIndex) > 0 {
			indexRandom := rand.Intn(len(subsIndex))
			valtoUnsub := subsIndex[indexRandom]
			newUnsubscription = GenerateUnsunscriptions(
				/*valIndexs*/ []uint64{valtoUnsub},
				/*valKeys*/ []string{"val" + strconv.FormatUint(valtoUnsub, 10)},
				/*sender*/ []string{strconv.FormatUint(50000+uint64(i), 10)},
				/*blockNums*/ []uint64{50000 + uint64(i)},
				/*txHashes*/ []string{"0x1"},
				/*depositAddrs*/ []string{strconv.FormatUint(50000+uint64(i), 10)},
			)
			//unsubsIndex = append(unsubsIndex, newUnsubscription[0].ValidatorIndex)

			//delete subbed validator from slice that keeps all subbed validators
			subsIndex = append(subsIndex[:indexRandom], subsIndex[indexRandom+1:]...)
		}
		//throw dice to determine if a new donation is set in this slot. 1/5 chance
		dice = rand.Intn(5)
		if dice == 0 {
			donationAmount := big.NewInt(int64(rand.Intn(1000) + 10000))
			newDonation := Donation{
				AmountWei: donationAmount,
				Block:     50000,
				TxHash:    "my_tx_hash",
			}
			don = append(don, newDonation)
			totalAssets.Add(totalAssets, donationAmount)
		}

		//throw dice to determine block type (ok, missed, wrongfee)
		dice = rand.Intn(3)
		//valToPropose := subsIndex[rand.Intn(len(subsIndex))]

		//choose randomly a validator to propopse the block (can be an unsubbed validator, so we check automatic subs)
		valToPropose := uint64(rand.Intn(numBlocks) + 50000)

		for _, sub := range newSubscription {
			log.WithFields(log.Fields{
				"ValidatorIndex":  sub.ValidatorIndex,
				"ValidatorKey":    sub.ValidatorKey,
				"Collateral":      sub.Collateral,
				"Deposit Address": sub.DepositAddress,
				"Tx Hash":         sub.TxHash,
			}).Info("Mocked Event: Subscription")
		}

		for _, unsub := range newUnsubscription {
			log.WithFields(log.Fields{
				"ValidatorIndex":  unsub.ValidatorIndex,
				"ValidatorKey":    unsub.ValidatorKey,
				"Sender":          unsub.Sender,
				"Deposit Address": unsub.DepositAddress,
				"Tx Hash":         unsub.TxHash,
			}).Info("Mocked Event: Unsubscription")
		}
		for _, don := range don {
			log.WithFields(log.Fields{
				"Amount(wei)": don.AmountWei,
				"Block":       don.Block,
				"Tx Hash":     don.TxHash,
			}).Info("Mocked Event: Donation")
		}

		log.Infof("Validator Index to propose: %d\n", valToPropose)
		if dice == 0 {
			log.Info("Block type: BlockOkProposal")
			mevReward := big.NewInt(int64(rand.Intn(1000) + 10000))
			processedSlot, err := oracle.AdvanceStateToNextSlot(blockOkProposal(
				50000+uint64(i),
				valToPropose,
				strconv.FormatUint(50000+uint64(i), 10),
				mevReward,
				"0xaaa0000000000000000000000000000000000000"), newSubscription, newUnsubscription, don)
			require.NoError(t, err)
			_ = processedSlot
			totalAssets.Add(totalAssets, mevReward) // block reward

		} else if dice == 1 {
			log.Info("Block type: MissedBlock")
			processedSlot, err := oracle.AdvanceStateToNextSlot(MissedBlock(
				50000+uint64(i),
				valToPropose,
				"0x"), newSubscription, newUnsubscription, don)
			require.NoError(t, err)
			_ = processedSlot

		} else {
			log.Info("Block type: WrongFeeBlock")
			processedSlot, err := oracle.AdvanceStateToNextSlot(WrongFeeBlock(
				50000+uint64(i),
				valToPropose,
				"0x"), newSubscription, newUnsubscription, don)
			require.NoError(t, err)
			_ = processedSlot

		}
	}

	// What we owe
	totalLiabilities := big.NewInt(0)
	for _, val := range oracle.State.Validators {
		totalLiabilities.Add(totalLiabilities, val.AccumulatedRewardsWei)
		totalLiabilities.Add(totalLiabilities, val.PendingRewardsWei)
	}
	totalLiabilities.Add(totalLiabilities, oracle.State.PoolAccumulatedFees) // TODO: rename wei

	require.Equal(t, totalAssets, totalLiabilities)
}

func Test_Oracle_WrongInputData(t *testing.T) {
}

func Test_Oracle_Unsubscription(t *testing.T) {
	unsubs := GenerateUnsunscriptions(
		/*valIndexs*/ []uint64{1, 2, 3},
		/*valKeys*/ []string{"0xaa", "0xba", "0xca"},
		/*senders*/ []string{"0xad", "0xbd", "0xcd"},
		/*blockNums*/ []uint64{0, 0, 0},
		/*txHashes*/ []string{"0xae", "0xbe", "0xce"},
		/*depositAddrs*/ []string{"0xaf", "0xbf", "0xcf"},
	)
	_ = unsubs
}

func Test_Oracle_Donation(t *testing.T) {
	blockWrongFee := Block{
		Slot: uint64(0), ValidatorIndex: uint64(1),
		ValidatorKey: "0xxx", BlockType: WrongFeeRecipient,
		Reward: big.NewInt(0), RewardType: MevBlock,
	}
	_ = blockWrongFee
}

func Test_Oracle_AutomaticSubscription(t *testing.T) {
	blockWrongFee := Block{
		Slot: uint64(0), ValidatorIndex: uint64(1),
		ValidatorKey: "0xxx", BlockType: WrongFeeRecipient,
		Reward: big.NewInt(0), RewardType: MevBlock,
	}
	_ = blockWrongFee
}

func Test_Oracle_WrongFee(t *testing.T) {
	blockWrongFee := Block{
		Slot: uint64(0), ValidatorIndex: uint64(1),
		ValidatorKey: "0xxx", BlockType: WrongFeeRecipient,
		Reward: big.NewInt(0), RewardType: MevBlock,
	}
	_ = blockWrongFee
}

func Test_Oracle_Missed_ToYellow(t *testing.T) {
	blockMissed := Block{
		Slot: uint64(0), ValidatorIndex: uint64(1),
		ValidatorKey: "0xxx", BlockType: MissedProposal,
	}

	_ = blockMissed

}

func Test_Oracle_Missed_ToRed(t *testing.T) {
	blockMissed := Block{
		Slot: uint64(0), ValidatorIndex: uint64(1),
		ValidatorKey: "0xxx", BlockType: MissedProposal,
	}

	_ = blockMissed

}

// Some util functions to faciliatet testing
func GenerateSubsctiptions(
	valIndex []uint64, valKey []string,
	collateral []*big.Int, blockNum []uint64,
	txHash []string, depAdd []string) []Subscription {

	subs := make([]Subscription, 0)

	for i := 0; i < len(valIndex); i++ {
		subs = append(subs, Subscription{
			ValidatorIndex: valIndex[i],
			ValidatorKey:   valKey[i],
			Collateral:     collateral[i],
			BlockNumber:    blockNum[i],
			TxHash:         txHash[i],
			DepositAddress: depAdd[i],
		})
	}
	return subs
}

func GenerateUnsunscriptions(
	valIndex []uint64, valKey []string,
	sender []string, blockNum []uint64,
	txHashes []string, depAdd []string) []Unsubscription {

	unsubs := make([]Unsubscription, 0)

	for i := 0; i < len(valIndex); i++ {
		unsubs = append(unsubs, Unsubscription{
			ValidatorIndex: valIndex[i],
			ValidatorKey:   valKey[i],
			Sender:         sender[i],
			BlockNumber:    blockNum[i],
			TxHash:         txHashes[i],
			DepositAddress: depAdd[i],
		})
	}
	return unsubs
}

func MissedBlock(slot uint64, valIndex uint64, pubKey string) Block {
	return Block{
		Slot:           slot,
		ValidatorIndex: valIndex,
		ValidatorKey:   pubKey,
		BlockType:      MissedProposal,
	}
}

func WrongFeeBlock(slot uint64, valIndex uint64, pubKey string) Block {
	return Block{
		Slot:           slot,
		ValidatorIndex: valIndex,
		ValidatorKey:   pubKey,
		BlockType:      WrongFeeRecipient,
	}
}

func blockOkProposal(slot uint64, valIndex uint64, pubKey string, reward *big.Int, depAddr string) Block {
	return Block{
		Slot:           slot,
		ValidatorIndex: valIndex,
		ValidatorKey:   pubKey,
		BlockType:      OkPoolProposal,
		Reward:         reward,
		RewardType:     MevBlock,
		DepositAddress: depAddr,
	}
}
