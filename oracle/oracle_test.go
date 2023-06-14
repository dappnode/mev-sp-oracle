package oracle

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// This file contains almost end to end tests, where the only mocked part is the
// data that is fetched onchain: blocks, subscriptions, unsubscriptions and donations

// TODO: Test merkle roots and proofs generation

// TODO:
func Test_Oracle_ManualSubscription(t *testing.T) {
	/*
		oracle := NewOracle(&Config{
			Network:               "",
			PoolAddress:           "0xdead000000000000000000000000000000000000",
			UpdaterAddress:        "",
			DeployedSlot:          uint64(50000),
			CheckPointSizeInSlots: uint64(100),
			PoolFeesPercent:       5,
			PoolFeesAddress:       "0xfee0000000000000000000000000000000000000",
			CollateralInWei:       big.NewInt(1000000),
		})

		subs := []Subscription{} // TODO:

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
			Reward: big.NewInt(245579896737171752), RewardType: MevBlock, WithdrawalAddress: "0xaaa0000000000000000000000000000000000000",
		}

		processedSlot, err = oracle.AdvanceStateToNextSlot(block1, []Subscription{}, []Unsubscription{}, []Donation{})
		require.NoError(t, err)
		require.Equal(t, uint64(50011), processedSlot)

		// Validator 500000 proposes a block
		block2 := Block{
			Slot: uint64(50012), ValidatorIndex: uint64(500000),
			ValidatorKey: "0xval_500000", BlockType: OkPoolProposal,
			Reward: big.NewInt(945579196337171700), RewardType: MevBlock, WithdrawalAddress: "0xaaa0000000000000000000000000000000000000",
		}

		processedSlot, err = oracle.AdvanceStateToNextSlot(block2, []Subscription{}, []Unsubscription{}, []Donation{})
		require.NoError(t, err)
		require.Equal(t, uint64(50012), processedSlot)

		enough := oracle.State.StoreLatestOnchainState()
		require.True(t, enough)

		require.Equal(t, "df67cc0d6a1d8b80f7d73b42813952c0e4d3936f597959fe87374eb89f100f5e", oracle.State.LatestCommitedState.MerkleRoot)

		// What we owe
		totalLiabilities := big.NewInt(0)
		for _, val := range oracle.state.Validators {
			totalLiabilities.Add(totalLiabilities, val.AccumulatedRewardsWei)
			totalLiabilities.Add(totalLiabilities, val.PendingRewardsWei)
		}
		totalLiabilities.Add(totalLiabilities, oracle.oracle.state.PoolAccumulatedFees) // TODO: rename wei

		// What we have (block fees + collateral)
		totalAssets := big.NewInt(0)
		totalAssets.Add(totalAssets, big.NewInt(245579896737171752)) // reward first block
		totalAssets.Add(totalAssets, big.NewInt(945579196337171700)) // reward second block
		for _, val := range oracle.state.Validators {
			totalAssets.Add(totalAssets, val.CollateralWei)
		}

		require.Equal(t, totalAssets, totalLiabilities)
	*/
}

// TODO: Mix manual and automatic subscriptions

// Simulates 100 slots with "AdvanceStateToNextSlot". Each slot is configured randomly with a
// new sub, unsub or donation. The block proposed can be okproposal, missed or wrongfee.
// these are all randomly set each block
/*
func Test_100_slots_test(t *testing.T) {
	numBlocks := 100
	log.Infof("Number of blocks to simulate: %d", numBlocks)
	//set new oracle instance
	oracle := NewOracle(&Config{
		Network:               "mainnet",
		PoolAddress:           "0xdead000000000000000000000000000000000000",
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

			//newSubscription = GenerateSubsctiptions(
			//	[]uint64{50000 + uint64(i)},
			//	[]string{"val" + strconv.FormatUint(50000+uint64(i), 10)},
			//	[]*big.Int{big.NewInt(1000000)},
			//	[]uint64{50000 + uint64(i)},
			//	[]string{"0x1"},
			//	[]string{"0xaaa0000000000000000000000000000000000000"},
			//)
			//subsIndex = append(subsIndex, newSubscription[0].ValidatorIndex)
			//totalAssets.Add(totalAssets, newSubscription[0].Collateral)
		}

		//throw dice to determine if a new unsubscription is set in this slot. 1/3 chance
		//(can only unsubscribe already subbed validators)
		dice = rand.Intn(3)
		if dice == 0 && len(subsIndex) > 0 {

			//indexRandom := rand.Intn(len(subsIndex))
			//valtoUnsub := subsIndex[indexRandom]
			//
			//newUnsubscription = GenerateUnsunscriptions(
			//	 []uint64{valtoUnsub},
			//	 []string{"val" + strconv.FormatUint(valtoUnsub, 10)},
			//	[]string{strconv.FormatUint(50000+uint64(i), 10)},
			//	 []uint64{50000 + uint64(i)},
			//	 []string{"0x1"},
			//	 []string{strconv.FormatUint(50000+uint64(i), 10)},
			//)
			////unsubsIndex = append(unsubsIndex, newUnsubscription[0].ValidatorIndex)
			//
			////delete subbed validator from slice that keeps all subbed validators
			//subsIndex = append(subsIndex[:indexRandom], subsIndex[indexRandom+1:]...)

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

		//for _, sub := range newSubscription {
		//	log.WithFields(log.Fields{
		//		"ValidatorIndex":  sub.ValidatorIndex,
		//		"ValidatorKey":    sub.ValidatorKey,
		//		"Collateral":      sub.Collateral,
		//		"withdrawal address": sub.WithdrawalAddress,
		//		"Tx Hash":         sub.TxHash,
		//	}).Info("Mocked Event: Subscription")
		//}
		//
		//for _, unsub := range newUnsubscription {
		//	log.WithFields(log.Fields{
		//		"ValidatorIndex":  unsub.ValidatorIndex,
		//		"ValidatorKey":    unsub.ValidatorKey,
		//		"Sender":          unsub.Sender,
		//		"withdrawal address": unsub.WithdrawalAddress,
		//		"Tx Hash":         unsub.TxHash,
		//	}).Info("Mocked Event: Unsubscription")
		//}
		//for _, don := range don {
		//	log.WithFields(log.Fields{
		//		"Amount(wei)": don.AmountWei,
		//		"Block":       don.Block,
		//		"Tx Hash":     don.TxHash,
		//	}).Info("Mocked Event: Donation")
		//}

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
	for _, val := range oracle.State().Validators {
		totalLiabilities.Add(totalLiabilities, val.AccumulatedRewardsWei)
		totalLiabilities.Add(totalLiabilities, val.PendingRewardsWei)
	}
	totalLiabilities.Add(totalLiabilities, oracle.State().PoolAccumulatedFees) // TODO: rename wei

	require.Equal(t, totalAssets, totalLiabilities)
}*/

func Test_Oracle_WrongInputData(t *testing.T) {
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

/*
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
			WithdrawalAddress: depAdd[i],
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
			WithdrawalAddress: depAdd[i],
		})
	}
	return unsubs
}*/

func Test_AddSubscription(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.AddSubscriptionIfNotAlready(10, "0x", "0x")
	oracle.IncreaseAllPendingRewards(big.NewInt(100))
	oracle.ConsolidateBalance(10)
	oracle.IncreaseAllPendingRewards(big.NewInt(200))
	require.Equal(t, big.NewInt(200), oracle.state.Validators[10].PendingRewardsWei)
	require.Equal(t, big.NewInt(100), oracle.state.Validators[10].AccumulatedRewardsWei)

	// check that adding again doesnt reset the subscription
	oracle.AddSubscriptionIfNotAlready(10, "0x", "0x")
	require.Equal(t, big.NewInt(200), oracle.state.Validators[10].PendingRewardsWei)
	require.Equal(t, big.NewInt(100), oracle.state.Validators[10].AccumulatedRewardsWei)
}

func Test_AddSubscriptionIfNotAlready(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.AddSubscriptionIfNotAlready(uint64(100), "0x3000000000000000000000000000000000000000", "0xkey")
	require.Equal(t, 1, len(oracle.state.Validators))
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(0),
		PendingRewardsWei:     big.NewInt(0),
		CollateralWei:         big.NewInt(0),
		WithdrawalAddress:     "0x3000000000000000000000000000000000000000",
		ValidatorIndex:        100,
		ValidatorKey:          "0xkey",
	}, oracle.state.Validators[100])

	// Modify the validator
	oracle.state.Validators[100].AccumulatedRewardsWei = big.NewInt(334545546)
	oracle.state.Validators[100].PendingRewardsWei = big.NewInt(87653)

	// If we call it again, it shouldnt be overwritten as its already there
	oracle.AddSubscriptionIfNotAlready(uint64(100), "0x3000000000000000000000000000000000000000", "0xkey")

	require.Equal(t, big.NewInt(334545546), oracle.state.Validators[100].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(87653), oracle.state.Validators[100].PendingRewardsWei)
}

func Test_AddDonation(t *testing.T) {
	oracle := NewOracle(&Config{})
	donations := []Donation{
		Donation{AmountWei: big.NewInt(765432), Block: uint64(100), TxHash: "0x1"},
		Donation{AmountWei: big.NewInt(30023456), Block: uint64(100), TxHash: "0x2"},
	}
	oracle.handleDonations(donations)

	require.Equal(t, big.NewInt(765432), oracle.state.Donations[0].AmountWei)
	require.Equal(t, uint64(100), oracle.state.Donations[0].Block)
	require.Equal(t, "0x1", oracle.state.Donations[0].TxHash)

	require.Equal(t, big.NewInt(30023456), oracle.state.Donations[1].AmountWei)
	require.Equal(t, uint64(100), oracle.state.Donations[1].Block)
	require.Equal(t, "0x2", oracle.state.Donations[1].TxHash)
}

// TODO: Merge all these tests into one
// TODO: test 2 ssubscriptions same block
func Test_handleManualSubscriptions_Valid(t *testing.T) {
	// Tests a valid subscription, with enough collateral to a not subscribed validator
	// and sent from the validator's withdrawal address

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		33: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	oracle.handleManualSubscriptions(subs)
	require.Equal(t, oracle.state.Validators[33], &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(0),
		PendingRewardsWei:     big.NewInt(1000),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        33,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	})
	require.Equal(t, 1, len(oracle.state.Validators))
	require.Equal(t, 1, len(oracle.state.Subscriptions))
	require.Equal(t, subs[0], oracle.state.Subscriptions[0])
}

func Test_handleManualSubscriptions_FromWrongAddress(t *testing.T) {
	// Tests a subscription sent from a wrong address, meaning that it doesnt
	// match the validator's withdrawal address. No subscription is produced

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		33: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	sub1 := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	// No subscriptions are produced
	oracle.handleManualSubscriptions(sub1)
	require.Equal(t, 0, len(oracle.state.Validators))
	require.Equal(t, 0, len(oracle.state.Subscriptions))
}

func Test_handleManualSubscriptions_AlreadySubscribed(t *testing.T) {
	// Test a subscription to an already subscribed validator, we return the collateral

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		33: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	// Run 3 subscriptions, only one should be added
	oracle.handleManualSubscriptions(subs)

	require.Equal(t, oracle.state.Validators[33], &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(2000), // Second and third collateral are returned to the user
		PendingRewardsWei:     big.NewInt(1000),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        33,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	})
	require.Equal(t, 1, len(oracle.state.Validators))
}

func Test_handleManualSubscriptions_AlreadySubscribed_WithBalance(t *testing.T) {
	// Test a subscription to an already subscribed validator, that already
	// has some balance. Assert that the existing balance is not touched and the
	// collateral is returned

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		33: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	sub1 := &contract.ContractSubscribeValidator{
		ValidatorID:            33,
		SubscriptionCollateral: big.NewInt(1000),
		Raw:                    types.Log{TxHash: [32]byte{0x1}},
		Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
	}

	// Validator is subscribed
	oracle.handleManualSubscriptions([]*contract.ContractSubscribeValidator{sub1})

	// And has some rewards
	oracle.IncreaseValidatorAccumulatedRewards(33, big.NewInt(9000))
	oracle.IncreaseValidatorPendingRewards(33, big.NewInt(44000))

	// Due to some mistake, the user subscribes again and again
	oracle.handleManualSubscriptions([]*contract.ContractSubscribeValidator{sub1, sub1})

	require.Equal(t, oracle.state.Validators[33], &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(9000 + 1000*2),
		PendingRewardsWei:     big.NewInt(44000 + 1000),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        33,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	})
	require.Equal(t, 1, len(oracle.state.Validators))
}

func Test_handleManualSubscriptions_Wrong_BlsCredentials(t *testing.T) {
	// A validator with wrong withdrawal address (bls) tries to subscribe. The validator
	// is nos subscribed and the collateral is given to the pool, since we dont have a way
	// to return it to its owner.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		33: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// bls address, not supported
				WithdrawalCredentials: []byte{0, 120, 22, 197, 153, 67, 183, 29, 244, 168, 13, 66, 101, 227, 165, 250, 41, 86, 97, 10, 40, 91, 140, 65, 154, 102, 143, 67, 117, 255, 140, 254},
			},
		},
	}

	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	oracle.handleManualSubscriptions(subs)
	require.Equal(t, 0, len(oracle.state.Validators))
	require.Equal(t, big.NewInt(1000), oracle.state.PoolAccumulatedFees)
}

func Test_handleManualSubscriptions_NonExistent(t *testing.T) {
	// Test a subscription of a non-existent validator. Someone subscribes a validator
	// index that doesnt exist. Nothing happens, and the pool gets this collateral.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		0: &v1.Validator{}, // dummuy validator
	}

	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	oracle.handleManualSubscriptions(subs)
	require.Equal(t, 0, len(oracle.state.Validators))
	require.Equal(t, big.NewInt(1000), oracle.state.PoolAccumulatedFees)
}

func Test_handleManualSubscriptions_WrongStateValidator(t *testing.T) {
	// Test a subscription of a validator in a wrong state (eg slashed validator or exited)
	// Nothing happens, and the pool gets this collateral.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		33: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateExitedSlashed,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
		},
		34: &v1.Validator{
			Index:  34,
			Status: v1.ValidatorStateActiveExiting,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
		},
	}

	subs := []*contract.ContractSubscribeValidator{
		// Its active but its exiting
		&contract.ContractSubscribeValidator{
			ValidatorID:            34,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x2}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		// Was slashed and exited
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	oracle.handleManualSubscriptions(subs)

	require.Equal(t, 0, len(oracle.state.Validators))
	require.Equal(t, big.NewInt(1000*2), oracle.state.PoolAccumulatedFees)
}

func Test_handleManualSubscriptions_BannedValidator(t *testing.T) {
	// Test a subscription of a banned validator. Check that the validator is not subscribed
	// and its kept in Banned state. Since we track this validator, we return the collateral
	// to the owner in good faith.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	bannedIndex := uint64(300000)

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		phase0.ValidatorIndex(bannedIndex): &v1.Validator{
			Index:  phase0.ValidatorIndex(bannedIndex),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
		},
	}

	oracle.state.Validators[bannedIndex] = &ValidatorInfo{
		ValidatorStatus:       Banned,
		AccumulatedRewardsWei: big.NewInt(0),
		PendingRewardsWei:     big.NewInt(0),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        bannedIndex,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}

	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            bannedIndex,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	oracle.handleManualSubscriptions(subs)

	// Banned validator stays banned
	require.Equal(t, 1, len(oracle.state.Validators))

	// Note that since we track it, we return the collateral as accumulated rewards
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:       Banned,
		AccumulatedRewardsWei: big.NewInt(1000),
		PendingRewardsWei:     big.NewInt(0),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        bannedIndex,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}, oracle.state.Validators[bannedIndex])
}

func Test_Handle_Subscriptions_1(t *testing.T) {

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		33: &v1.Validator{
			Index:  33,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},

		34: &v1.Validator{
			Index:  34,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 149, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{130, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},

		35: &v1.Validator{
			Index:  35,
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 150, 39, 165, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
				PublicKey: phase0.BLSPubKey{131, 170, 2, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	//Set up 3 new subs (val index 33,34,35), two valid and one invalid (low collateral)
	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		&contract.ContractSubscribeValidator{
			ValidatorID:            34,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{149, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
		&contract.ContractSubscribeValidator{
			ValidatorID:            35,
			SubscriptionCollateral: big.NewInt(50),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{150, 39, 165, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	oracle.handleManualSubscriptions(subs)

	// 3 validator tried to sub, 2 ok, 1 not enough collateral
	require.Equal(t, 2, len(oracle.state.Validators))
	require.Equal(t, 2, len(oracle.state.Subscriptions))

	require.Equal(t, subs[0], oracle.state.Subscriptions[0])

	//one validator subscribed with wrong collateral --> sent to the pool
	require.Equal(t, big.NewInt(50), oracle.state.PoolAccumulatedFees)

	// We keep track of [33 & 34] since subscription was valid
	require.Equal(t, Active, oracle.state.Validators[33].ValidatorStatus)
	require.Equal(t, Active, oracle.state.Validators[34].ValidatorStatus)

	// Accumulated rewards should be 0
	require.Equal(t, big.NewInt(0), oracle.state.Validators[33].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(0), oracle.state.Validators[34].AccumulatedRewardsWei)

	// Collateral should be 1000
	require.Equal(t, big.NewInt(1000), oracle.state.Validators[33].CollateralWei)
	require.Equal(t, big.NewInt(1000), oracle.state.Validators[34].CollateralWei)

	//Set up 2 new subs, both of already subscribed validators one sends configured collateral, the other does not
	subs2 := []*contract.ContractSubscribeValidator{
		// validator already subscribed sends subscription again with too much collateral
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(5000000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},

		// validator already subscribed sends subscription again with correct collateral
		&contract.ContractSubscribeValidator{
			ValidatorID:            34,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{149, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	oracle.handleManualSubscriptions(subs2)

	// [33] & [34] should still be active after trying to subscribe again
	require.Equal(t, Active, oracle.state.Validators[33].ValidatorStatus)
	require.Equal(t, Active, oracle.state.Validators[34].ValidatorStatus)

	// when an already subscribed validator manually subscribes again, we send the collateral to their accumulated rewards
	require.Equal(t, big.NewInt(5000000), oracle.state.Validators[33].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(1000), oracle.state.Validators[34].AccumulatedRewardsWei)

	// Collateral does not change
	require.Equal(t, big.NewInt(1000), oracle.state.Validators[33].CollateralWei)
	require.Equal(t, big.NewInt(1000), oracle.state.Validators[34].CollateralWei)

	// Ban validator 34
	oracle.handleBanValidator(Block{
		Slot:           uint64(100),
		ValidatorIndex: uint64(34),
	})

	// Validator 34 should be banned
	require.Equal(t, Banned, oracle.state.Validators[34].ValidatorStatus)
	// Accumulated does not change
	require.Equal(t, big.NewInt(1000), oracle.state.Validators[34].AccumulatedRewardsWei)

	// Accumulated rewards of other validators does not change because banned validator didnt have pending rewards
	require.Equal(t, big.NewInt(5000000), oracle.state.Validators[33].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(1000), oracle.state.Validators[34].AccumulatedRewardsWei)
}

func Test_SubThenUnsubThenAuto(t *testing.T) {

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(500000),
		PoolFeesPercent: 0,
	})

	// Subscribe a validator
	valIdx := uint64(9000)

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		phase0.ValidatorIndex(valIdx): &v1.Validator{
			Index:  phase0.ValidatorIndex(valIdx),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{7, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            valIdx,
			SubscriptionCollateral: big.NewInt(500000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}
	oracle.handleManualSubscriptions(subs)

	// Share some rewards with it
	oracle.state.Validators[valIdx].PendingRewardsWei = big.NewInt(10000)
	oracle.state.Validators[valIdx].AccumulatedRewardsWei = big.NewInt(20000)

	// Unsubscribe it
	unsubs := []*contract.ContractUnsubscribeValidator{
		&contract.ContractUnsubscribeValidator{
			ValidatorID: valIdx,
			Sender:      common.Address{3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:         types.Log{TxHash: [32]byte{0x1}},
		},
	}

	oracle.handleManualUnsubscriptions(unsubs)

	// Check is no longer subscribed and balances are kept (pending is reset)
	require.Equal(t, big.NewInt(0), oracle.state.Validators[valIdx].PendingRewardsWei)
	require.Equal(t, big.NewInt(20000), oracle.state.Validators[valIdx].AccumulatedRewardsWei)
	require.Equal(t, NotSubscribed, oracle.state.Validators[valIdx].ValidatorStatus)

	// Force automatic subscription
	block1 := Block{
		Slot:              0,
		ValidatorIndex:    valIdx,
		ValidatorKey:      "0x",
		Reward:            big.NewInt(90000000),
		RewardType:        VanilaBlock,
		WithdrawalAddress: "0x0327a30991170f917d7b83def6e44d26577871ed",
	}
	oracle.handleCorrectBlockProposal(block1)

	// Pending are 0 because the 90000000 are instantly consolidated
	require.Equal(t, big.NewInt(0), oracle.state.Validators[valIdx].PendingRewardsWei)
	// We have the new plus old ones
	require.Equal(t, big.NewInt(20000+90000000), oracle.state.Validators[valIdx].AccumulatedRewardsWei)
	require.Equal(t, Active, oracle.state.Validators[valIdx].ValidatorStatus)
}

func Test_HandleUnsubscriptions_ValidSubscription(t *testing.T) {
	// Unsubscribe an existing subscribed validator correctly, checking that the event is
	// sent from the withdrawal address of the validator. Check also that when unsubscribing
	// the pending validator rewards are shared among the rest of the validators.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(500000),
	})
	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{}
	for _, valIdx := range []uint64{6, 9, 10, 15} {
		oracle.beaconValidators[phase0.ValidatorIndex(valIdx)] = &v1.Validator{
			Index:  phase0.ValidatorIndex(valIdx),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// byte(valIdx) just to have different key/withdrawal addresses
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{byte(valIdx), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		}
	}

	for _, valIdx := range []uint64{6, 9, 10, 15} {
		subs := []*contract.ContractSubscribeValidator{
			&contract.ContractSubscribeValidator{
				ValidatorID:            valIdx,
				SubscriptionCollateral: big.NewInt(500000),
				Raw:                    types.Log{TxHash: [32]byte{0x1}},
				Sender:                 common.Address{byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			},
		}
		oracle.handleManualSubscriptions(subs)

		// Simulate some proposals increasing the rewards
		oracle.IncreaseValidatorAccumulatedRewards(valIdx, big.NewInt(3000))
		oracle.IncreaseValidatorPendingRewards(valIdx, big.NewInt(300000000000000000-500000))
	}

	require.Equal(t, 4, len(oracle.state.Validators))

	// Receive valid unsubscription event for index 6
	unsubs := []*contract.ContractUnsubscribeValidator{
		&contract.ContractUnsubscribeValidator{
			ValidatorID: 6,
			// Same as withdrawal credential without the prefix
			Sender: common.Address{byte(6), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
	}
	oracle.handleManualUnsubscriptions(unsubs)

	require.Equal(t, oracle.state.Validators[6], &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,    // Validator is still tracked but not subscribed
		AccumulatedRewardsWei: big.NewInt(3000), // Accumulated rewards are kept
		PendingRewardsWei:     big.NewInt(0),    // Pending rewards are cleared
		CollateralWei:         big.NewInt(500000),
		WithdrawalAddress:     "0x0627a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        6,
		ValidatorKey:          "0x06aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	})
	require.Equal(t, 4, len(oracle.state.Validators))
	require.Equal(t, 1, len(oracle.state.Unsubscriptions))
	require.Equal(t, unsubs[0], oracle.state.Unsubscriptions[0])

	// The rest get the pending of valIndex=6
	require.Equal(t, oracle.state.Validators[9].PendingRewardsWei, big.NewInt(300000000000000000+300000000000000000/3))
	require.Equal(t, oracle.state.Validators[10].PendingRewardsWei, big.NewInt(300000000000000000+300000000000000000/3))
	require.Equal(t, oracle.state.Validators[15].PendingRewardsWei, big.NewInt(300000000000000000+300000000000000000/3))

	// And accumulated do not change
	require.Equal(t, oracle.state.Validators[9].AccumulatedRewardsWei, big.NewInt(3000))
	require.Equal(t, oracle.state.Validators[10].AccumulatedRewardsWei, big.NewInt(3000))
	require.Equal(t, oracle.state.Validators[15].AccumulatedRewardsWei, big.NewInt(3000))

	// And state of the rest is not changed
	require.Equal(t, oracle.state.Validators[9].ValidatorStatus, Active)
	require.Equal(t, oracle.state.Validators[10].ValidatorStatus, Active)
	require.Equal(t, oracle.state.Validators[15].ValidatorStatus, Active)

	// Unsubscribe all remaining validators
	newUnsubs := make([]*contract.ContractUnsubscribeValidator, 0)
	for _, valIdx := range []uint64{ /*6*/ 9, 10, 15} {
		newUnsubs = append(newUnsubs,
			&contract.ContractUnsubscribeValidator{
				ValidatorID: valIdx,
				// Same as withdrawal credential without the prefix
				Sender: common.Address{byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				Raw:    types.Log{TxHash: [32]byte{0x1}},
			})
	}

	// Unsubscribe all at once
	oracle.handleManualUnsubscriptions(newUnsubs)

	require.Equal(t, 4, len(oracle.state.Validators))
	require.Equal(t, oracle.state.Validators[6].ValidatorStatus, NotSubscribed)
	require.Equal(t, oracle.state.Validators[9].ValidatorStatus, NotSubscribed)
	require.Equal(t, oracle.state.Validators[10].ValidatorStatus, NotSubscribed)
	require.Equal(t, oracle.state.Validators[15].ValidatorStatus, NotSubscribed)

	require.Equal(t, oracle.state.Validators[6].PendingRewardsWei, big.NewInt(0))
	require.Equal(t, oracle.state.Validators[9].PendingRewardsWei, big.NewInt(0))
	require.Equal(t, oracle.state.Validators[10].PendingRewardsWei, big.NewInt(0))
	require.Equal(t, oracle.state.Validators[15].PendingRewardsWei, big.NewInt(0))
}

func Test_HandleUnsubscriptions_NonExistentValidator(t *testing.T) {
	// We receive an unsubscription for a validator that does not exist in the beacon
	// chain. Nothing happens to existing subscribed validators.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		0: &v1.Validator{}, // dummy validator
	}

	// Simulate subscription of validator 33
	oracle.state.Validators[33] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(9000 + 1000*2), // Second and third collateral are added to accumulated rewards (returned)
		PendingRewardsWei:     big.NewInt(44000 + 1000),  // First collateral is added to pending (claimable in next block)
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        33,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}

	// Receive event of a validator index that doesnt exist in the beacon chain
	unsubs := []*contract.ContractUnsubscribeValidator{
		&contract.ContractUnsubscribeValidator{
			ValidatorID: 900300,
			// Same as withdrawal credential without the prefix
			Sender: common.Address{byte(50), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
	}
	oracle.handleManualUnsubscriptions(unsubs)

	// Check that the existing validator is not affected
	require.Equal(t, 1, len(oracle.state.Validators))
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(9000 + 1000*2), // Second and third collateral are added to accumulated rewards (returned)
		PendingRewardsWei:     big.NewInt(44000 + 1000),  // First collateral is added to pending (claimable in next block)
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        33,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}, oracle.state.Validators[33])
}

func Test_HandleUnsubscriptions_NotSubscribedValidator(t *testing.T) {
	// We receive an unsubscription for a validator that is not subscribed but exists in
	// the beacon chain. Nothing happens, and no subscriptions are added.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		0: &v1.Validator{}, // dummy validator
	}

	// Unsubscribe event of a validator index that BUT is not subscribed
	valIdx := uint64(730100)
	unsubs := []*contract.ContractUnsubscribeValidator{
		&contract.ContractUnsubscribeValidator{
			ValidatorID: valIdx,
			// Same as withdrawal credential without the prefix
			Sender: common.Address{byte(valIdx), 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
	}
	oracle.handleManualUnsubscriptions(unsubs)
	require.Equal(t, 0, len(oracle.state.Validators))
}

func Test_HandleUnsubscriptions_FromWrongAddress(t *testing.T) {
	// An unsubscription for a subscribed validator is received, but the sender is not the
	// withdrawal address of that validator. Nothing happens to this validator

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		0: &v1.Validator{}, // dummy validator
	}

	// Simulate subscription of validator 750100
	valIndex := uint64(750100)
	oracle.state.Validators[valIndex] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(5000000000000000000),
		PendingRewardsWei:     big.NewInt(3000000000000000000),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        valIndex,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}

	unsubs := []*contract.ContractUnsubscribeValidator{
		&contract.ContractUnsubscribeValidator{
			ValidatorID: valIndex,
			// Wrong sender address (see WithdrawalCredentials)
			Sender: common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
	}
	oracle.handleManualUnsubscriptions(unsubs)

	// Validator remains intact, since unsubscription event was wrong
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(5000000000000000000),
		PendingRewardsWei:     big.NewInt(3000000000000000000),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        valIndex,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}, oracle.state.Validators[valIndex])
}

func Test_Unsubscribe_AndRejoin(t *testing.T) {
	// A validator subscribes, the unsubscribes and the rejoins. Check that its accumulated balances
	// are kept, and that it can rejoin succesfully.

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(500000),
	})

	valIndex := uint64(750100)

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
		phase0.ValidatorIndex(valIndex): &v1.Validator{
			Index:  phase0.ValidatorIndex(valIndex),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// byte(valIdx) just to have different key
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
				PublicKey:             phase0.BLSPubKey{byte(valIndex), 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
			},
		},
	}

	// Simulate subscription of validator 750100

	oracle.state.Validators[valIndex] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(0),
		PendingRewardsWei:     big.NewInt(0),
		CollateralWei:         big.NewInt(500000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        valIndex,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}

	// Add some rewards
	oracle.IncreaseValidatorAccumulatedRewards(valIndex, big.NewInt(1000000000000000000))
	oracle.IncreaseValidatorPendingRewards(valIndex, big.NewInt(5000000000000000000))

	// Now it unsubscribes ok
	unsubs := []*contract.ContractUnsubscribeValidator{
		&contract.ContractUnsubscribeValidator{
			ValidatorID: valIndex,
			// Wrong sender address (see WithdrawalCredentials)
			Sender: common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
			Raw:    types.Log{TxHash: [32]byte{0x1}},
		},
	}
	oracle.handleManualUnsubscriptions(unsubs)

	// Unsubscription is ok
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,
		AccumulatedRewardsWei: big.NewInt(1000000000000000000),
		PendingRewardsWei:     big.NewInt(0),
		CollateralWei:         big.NewInt(500000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        valIndex,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}, oracle.state.Validators[valIndex])

	// Now the same validator tries to rejoin
	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            valIndex,
			SubscriptionCollateral: big.NewInt(500000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}
	oracle.handleManualSubscriptions(subs)

	// Its subscribed again with its old accumulated rewards
	require.Equal(t, &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(1000000000000000000),
		PendingRewardsWei:     big.NewInt(500000),
		CollateralWei:         big.NewInt(500000),
		WithdrawalAddress:     "0x9427a30991170f917d7b83def6e44d26577871ed",
		ValidatorIndex:        valIndex,
		ValidatorKey:          "0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d",
	}, oracle.state.Validators[valIndex])
}

func Test_StoreLatestOnchainState(t *testing.T) {

	oracle := NewOracle(&Config{
		PoolFeesPercent: 0,
		PoolFeesAddress: "0xfee0000000000000000000000000000000000000",
	})

	valInfo1 := &ValidatorInfo{
		ValidatorStatus:       Active,
		ValidatorIndex:        1,
		AccumulatedRewardsWei: big.NewInt(1000000000000000000),
		PendingRewardsWei:     big.NewInt(500000),
		WithdrawalAddress:     "0x1000000000000000000000000000000000000000",
	}

	valInfo2 := &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,
		ValidatorIndex:        2,
		AccumulatedRewardsWei: big.NewInt(2000000000000000000),
		PendingRewardsWei:     big.NewInt(500000),
		// same withdrawal address as valInfo3
		WithdrawalAddress: "0x2000000000000000000000000000000000000000",
	}

	valInfo3 := &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,
		ValidatorIndex:        3,
		AccumulatedRewardsWei: big.NewInt(2000000000000000000),
		PendingRewardsWei:     big.NewInt(500000),
		// same withdrawal address as valInfo2
		WithdrawalAddress: "0x2000000000000000000000000000000000000000",
	}

	oracle.state.Validators[1] = valInfo1
	oracle.state.Validators[2] = valInfo2
	oracle.state.Validators[3] = valInfo3

	// Function under test
	oracle.StoreLatestOnchainState()

	// Ensure all validators are present in the state
	require.Equal(t, valInfo1, oracle.state.LatestCommitedState.Validators[1])
	require.Equal(t, valInfo2, oracle.state.LatestCommitedState.Validators[2])
	require.Equal(t, valInfo3, oracle.state.LatestCommitedState.Validators[3])

	// Ensure merkle root matches
	require.Equal(t, "0xd9a1eee574026532cddccbcce6320c0600f370a7c64ce30c5eafc63357449940", oracle.state.LatestCommitedState.MerkleRoot)

	// Ensure proofs and leafs are correct
	require.Equal(t, oracle.state.LatestCommitedState.Proofs["0xfee0000000000000000000000000000000000000"], []string{"0x8bfb8acff6772a60d6641cb854587bb2b6f2100391fbadff2c34be0b8c20a0cc", "0x27205dd4c642acd1b1352617df2c4f410e20ff3fd6f3e3efddee9cea044921f8"})
	require.Equal(t, oracle.state.LatestCommitedState.Proofs["0x1000000000000000000000000000000000000000"], []string{"0xaaf838df9c8d5cec6ed77fcbc2cace945e8f2078eede4a0bb7164818d425f24d", "0x27205dd4c642acd1b1352617df2c4f410e20ff3fd6f3e3efddee9cea044921f8"})
	require.Equal(t, oracle.state.LatestCommitedState.Proofs["0x2000000000000000000000000000000000000000"], []string{"0xd643163144dcba353b4d27c50939b3d11133bd3c6916092de059d07353b4cb5f", "0xda53f5dd3e17f66f4a35c9c9d5fd27c094fa4249e2933fb819ac724476dc9ae1"})

	require.Equal(t, oracle.state.LatestCommitedState.Leafs["0xfee0000000000000000000000000000000000000"], RawLeaf{"0xfee0000000000000000000000000000000000000", big.NewInt(0)})
	require.Equal(t, oracle.state.LatestCommitedState.Leafs["0x1000000000000000000000000000000000000000"], RawLeaf{"0x1000000000000000000000000000000000000000", big.NewInt(1000000000000000000)})
	require.Equal(t, oracle.state.LatestCommitedState.Leafs["0x2000000000000000000000000000000000000000"], RawLeaf{"0x2000000000000000000000000000000000000000", big.NewInt(4000000000000000000)})

	// Ensure LatestCommitedState contains a deep copy of the validators and not just a reference
	// This is very important since otherwise they will be modified when the state is modified
	// and we want a frozen snapshot of the state at that moment.

	// Do some changes in validators
	oracle.state.Validators[2].AccumulatedRewardsWei = big.NewInt(22)
	oracle.state.Validators[3].PendingRewardsWei = big.NewInt(22)

	// And assert the frozen state is not changes
	require.Equal(t, big.NewInt(2000000000000000000), oracle.state.LatestCommitedState.Validators[2].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(500000), oracle.state.LatestCommitedState.Validators[3].PendingRewardsWei)
}

func Test_IncreaseAllPendingRewards_1(t *testing.T) {

	oracle := NewOracle(&Config{
		PoolFeesPercent: 0,
		PoolFeesAddress: "0x",
	})

	// Subscribe 3 validators with no balance
	oracle.AddSubscriptionIfNotAlready(1, "0x", "0x")
	oracle.AddSubscriptionIfNotAlready(2, "0x", "0x")
	oracle.AddSubscriptionIfNotAlready(3, "0x", "0x")

	oracle.IncreaseAllPendingRewards(big.NewInt(10000))

	// Note that in this case even with PoolFeesPercent: 0, the pool gets the remainder
	require.Equal(t, big.NewInt(3333), oracle.state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(3333), oracle.state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(3333), oracle.state.Validators[3].PendingRewardsWei)
	require.Equal(t, big.NewInt(1), oracle.state.PoolAccumulatedFees)
}

func Test_IncreaseAllPendingRewards_2(t *testing.T) {

	oracle := NewOracle(&Config{
		PoolFeesPercent: 10 * 100, // 10%
		PoolFeesAddress: "0x",
	})

	// Subscribe 3 validators with no balance
	oracle.AddSubscriptionIfNotAlready(1, "0x", "0x")
	oracle.AddSubscriptionIfNotAlready(2, "0x", "0x")
	oracle.AddSubscriptionIfNotAlready(3, "0x", "0x")

	oracle.IncreaseAllPendingRewards(big.NewInt(10000))

	// Note that in this case even with PoolFeesPercent: 0, the pool gets the remainder
	require.Equal(t, big.NewInt(3000), oracle.state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(3000), oracle.state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(3000), oracle.state.Validators[3].PendingRewardsWei)
	require.Equal(t, big.NewInt(1000), oracle.state.PoolAccumulatedFees)
}

func Test_IncreaseAllPendingRewards_3(t *testing.T) {

	// Multiple test with different combinations of: fee, reward, validators

	type pendingRewardTest struct {
		FeePercent       int
		Reward           []*big.Int
		AmountValidators int
	}

	tests := []pendingRewardTest{
		// FeePercent |Reward | AmountValidators
		{0, []*big.Int{big.NewInt(100)}, 1},
		{0, []*big.Int{big.NewInt(500)}, 2},
		{0, []*big.Int{big.NewInt(398)}, 3},
		{10 * 100, []*big.Int{big.NewInt(0)}, 1},
		{15 * 100, []*big.Int{big.NewInt(23033)}, 1},
		{33 * 100, []*big.Int{big.NewInt(99999)}, 5},
		{33 * 100, []*big.Int{big.NewInt(1)}, 5},
		{33 * 100, []*big.Int{big.NewInt(1), big.NewInt(403342)}, 200},
		{12 * 100, []*big.Int{big.NewInt(32000000000000), big.NewInt(333333333333), big.NewInt(345676543234567)}, 233},
		{14 * 100, []*big.Int{big.NewInt(32000000000000), big.NewInt(333333333333), big.NewInt(345676543234567), big.NewInt(9)}, 99},
	}

	for _, test := range tests {
		oracle := NewOracle(&Config{
			PoolFeesPercent: test.FeePercent,
			PoolFeesAddress: "0x",
		})

		for i := 0; i < test.AmountValidators; i++ {
			oracle.AddSubscriptionIfNotAlready(uint64(i), "0x", "0x")
		}

		totalRewards := big.NewInt(0)
		for _, reward := range test.Reward {
			oracle.IncreaseAllPendingRewards(reward)
			totalRewards.Add(totalRewards, reward)
		}

		totalDistributedRewards := big.NewInt(0)
		totalDistributedRewards.Add(totalDistributedRewards, oracle.state.PoolAccumulatedFees)
		for i := 0; i < test.AmountValidators; i++ {
			totalDistributedRewards.Add(totalDistributedRewards, oracle.state.Validators[uint64(i)].PendingRewardsWei)
		}

		// Assert that the rewards that were shared, equal the ones that we had
		// kirchhoff law, what comes in = what it goes out!
		require.Equal(t, totalDistributedRewards, totalRewards)
	}
}

func Test_IncreaseAllPendingRewards_4(t *testing.T) {

	// Multiple test with different combinations of: fee, reward, validators

	type pendingRewardTest struct {
		FeePercentX100   int
		Reward           *big.Int
		AmountValidators int
		NewPendingArray  []*big.Int
		FeesAddress      *big.Int
	}

	tests := []pendingRewardTest{
		// FeePercentX100 (100 = 1%) | Reward | AmountValidators | RewardsPerValidator

		// 0%
		{0, big.NewInt(0), 1, []*big.Int{big.NewInt(0)}, big.NewInt(0)},

		// 100%
		{100 * 100, big.NewInt(2345676543), 0, []*big.Int{big.NewInt(0)}, big.NewInt(2345676543)},

		// 1%
		{1 * 100, big.NewInt(100), 1, []*big.Int{big.NewInt(99)}, big.NewInt(1)},

		// 10%
		{10 * 100, big.NewInt(10000000000), 1, []*big.Int{big.NewInt(9000000000)}, big.NewInt(1000000000)},

		// 10 % with 2 validators
		{10 * 100, big.NewInt(10000000000), 2, []*big.Int{big.NewInt(4500000000), big.NewInt(4500000000)}, big.NewInt(1000000000)},

		// 0%
		{0, big.NewInt(555555555555), 5, []*big.Int{big.NewInt(111111111111), big.NewInt(111111111111), big.NewInt(111111111111), big.NewInt(111111111111), big.NewInt(111111111111)}, big.NewInt(0)},

		// 2.5%
		{2.5 * 100, big.NewInt(15000), 5, []*big.Int{big.NewInt(2925), big.NewInt(2925), big.NewInt(2925), big.NewInt(2925), big.NewInt(2925)}, big.NewInt(375)},

		// 0.25%: 87654567898 * 25 / 10000 = 219136419 (+ remainder of 7450). Then 887654567898-(219136419 + 7450))/5 = 17487084805 (+ reminder of 4)
		{0.25 * 100, big.NewInt(87654567898), 5, []*big.Int{big.NewInt(17487084805), big.NewInt(17487084805), big.NewInt(17487084805), big.NewInt(17487084805), big.NewInt(17487084805)}, big.NewInt(219136419 + 7450 + 4)},
	}

	for _, test := range tests {
		oracle := NewOracle(&Config{
			PoolFeesPercent: test.FeePercentX100,
			PoolFeesAddress: "0x",
		})
		for i := 0; i < test.AmountValidators; i++ {
			oracle.AddSubscriptionIfNotAlready(uint64(i), "0x", "0x")
		}
		oracle.IncreaseAllPendingRewards(test.Reward)
		for i := 0; i < test.AmountValidators; i++ {
			require.Equal(t, test.NewPendingArray[i], oracle.state.Validators[uint64(i)].PendingRewardsWei)
		}

		require.Equal(t, test.FeesAddress, oracle.state.PoolAccumulatedFees)

		// Ensure that what we gave away matches the reward we had
		totalDistributedRewards := big.NewInt(0)
		totalDistributedRewards.Add(totalDistributedRewards, oracle.state.PoolAccumulatedFees)

		// Since all validators rewards are equal, just take the reward of the first one
		rewardsToValidators := big.NewInt(0)
		if len(oracle.state.Validators) > 0 {
			rewardsToValidators = oracle.state.Validators[0].PendingRewardsWei
		}
		rewardsToValidators.Mul(rewardsToValidators, big.NewInt(int64(test.AmountValidators)))
		totalDistributedRewards.Add(totalDistributedRewards, rewardsToValidators)
		require.Equal(t, totalDistributedRewards, test.Reward)
	}
}

func Test_IncreaseValidatorPendingRewards(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.state.Validators[12] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(100),
		AccumulatedRewardsWei: big.NewInt(0),
	}
	oracle.state.Validators[200] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(100),
		AccumulatedRewardsWei: big.NewInt(0),
	}

	oracle.IncreaseValidatorPendingRewards(12, big.NewInt(8765432))
	require.Equal(t, big.NewInt(8765432+100), oracle.state.Validators[12].PendingRewardsWei)
	require.Equal(t, big.NewInt(0), oracle.state.Validators[12].AccumulatedRewardsWei)

	oracle.IncreaseValidatorPendingRewards(200, big.NewInt(0))
	require.Equal(t, big.NewInt(100), oracle.state.Validators[200].PendingRewardsWei)

	oracle.IncreaseValidatorPendingRewards(12, big.NewInt(1))
	require.Equal(t, big.NewInt(8765432+100+1), oracle.state.Validators[12].PendingRewardsWei)
}

func Test_IncreaseValidatorAccumulatedRewards(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.state.Validators[9999999] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(100),
		AccumulatedRewardsWei: big.NewInt(99999999999999),
	}
	oracle.IncreaseValidatorAccumulatedRewards(9999999, big.NewInt(87676545432))
	require.Equal(t, big.NewInt(87676545432+99999999999999), oracle.state.Validators[9999999].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(100), oracle.state.Validators[9999999].PendingRewardsWei)
}

func Test_SendRewardToPool(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.SendRewardToPool(big.NewInt(10456543212340))
	require.Equal(t, big.NewInt(10456543212340), oracle.state.PoolAccumulatedFees)

	oracle.SendRewardToPool(big.NewInt(99999))
	require.Equal(t, big.NewInt(10456543212340+99999), oracle.state.PoolAccumulatedFees)
}

func Test_ResetPendingRewards(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.state.Validators[1] = &ValidatorInfo{
		PendingRewardsWei:     big.NewInt(99999999999999),
		AccumulatedRewardsWei: big.NewInt(99999999999999),
	}
	oracle.ResetPendingRewards(1)

	require.Equal(t, big.NewInt(0), oracle.state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(99999999999999), oracle.state.Validators[1].AccumulatedRewardsWei)
}

func Test_IncreasePendingRewards(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.state.Validators[12] = &ValidatorInfo{
		WithdrawalAddress: "0xaa",
		ValidatorStatus:   Active,
		PendingRewardsWei: big.NewInt(100),
	}
	totalAmount := big.NewInt(130)

	require.Equal(t, big.NewInt(100), oracle.state.Validators[12].PendingRewardsWei)
	oracle.IncreaseAllPendingRewards(totalAmount)
	require.Equal(t, big.NewInt(230), oracle.state.Validators[12].PendingRewardsWei)
}

func Test_IncreasePendingEmptyPool(t *testing.T) {
	// Test a case where a new rewards adds to the pool but no validators are subscribed
	// This can happen when a donation is recived to the pool but no validators are subscribed
	oracle := NewOracle(&Config{})

	// This prevents division by zero
	oracle.IncreaseAllPendingRewards(big.NewInt(10000))

	// Pool gets all rewards
	require.Equal(t, big.NewInt(10000), oracle.state.PoolAccumulatedFees)
}

func Test_ConsolidateBalance_Eligible(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.state.Validators[10] = &ValidatorInfo{
		AccumulatedRewardsWei: big.NewInt(77),
		PendingRewardsWei:     big.NewInt(23),
	}

	require.Equal(t, big.NewInt(77), oracle.state.Validators[10].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(23), oracle.state.Validators[10].PendingRewardsWei)

	oracle.ConsolidateBalance(10)

	require.Equal(t, big.NewInt(100), oracle.state.Validators[10].AccumulatedRewardsWei)
	require.Equal(t, big.NewInt(0), oracle.state.Validators[10].PendingRewardsWei)
}

func Test_StateMachine(t *testing.T) {
	oracle := NewOracle(&Config{})
	valIndex1 := uint64(1000)
	valIndex2 := uint64(2000)

	type stateTest struct {
		From  ValidatorStatus
		Event Event
		End   ValidatorStatus
	}

	stateMachineTestVector := []stateTest{
		// FromState | Event | EndState
		{Active, ProposalOk, Active},
		{Active, ProposalMissed, YellowCard},
		{Active, ProposalWrongFee, Banned},
		{Active, Unsubscribe, NotSubscribed},

		{YellowCard, ProposalOk, Active},
		{YellowCard, ProposalMissed, RedCard},
		{YellowCard, ProposalWrongFee, Banned},
		{YellowCard, Unsubscribe, NotSubscribed},

		{RedCard, ProposalOk, YellowCard},
		{RedCard, ProposalMissed, RedCard},
		{RedCard, ProposalWrongFee, Banned},
		{RedCard, Unsubscribe, NotSubscribed},

		{NotSubscribed, ManualSubscription, Active},
		{NotSubscribed, AutoSubscription, Active},
	}

	for _, testState := range stateMachineTestVector {
		oracle.state.Validators[valIndex1] = &ValidatorInfo{
			ValidatorStatus: testState.From,
		}
		oracle.state.Validators[valIndex2] = &ValidatorInfo{
			ValidatorStatus: testState.From,
		}

		oracle.AdvanceStateMachine(valIndex1, testState.Event)
		oracle.AdvanceStateMachine(valIndex2, testState.Event)

		require.Equal(t, testState.End, oracle.state.Validators[valIndex1].ValidatorStatus)
		require.Equal(t, testState.End, oracle.state.Validators[valIndex2].ValidatorStatus)
	}
}

// TODO: Test that if the file changes it fails due to hash
func Test_SaveReadToFromJson(t *testing.T) {
	oracle := NewOracle(&Config{
		PoolAddress:     "0x0000000000000000000000000000000000000000",
		PoolFeesAddress: "0x1000000000000000000000000000000000000000",
		Network:         "mainnet",
	})

	oracle.AddSubscriptionIfNotAlready(uint64(3), "0x1000000000000000000000000000000000000000", "0x1000000000000000000000000000000000000000")
	oracle.AddSubscriptionIfNotAlready(uint64(6434), "0x2000000000000000000000000000000000000000", "0x2000000000000000000000000000000000000000")

	oracle.StoreLatestOnchainState()

	oracle.AddSubscriptionIfNotAlready(uint64(3), "0x1000000000000000000000000000000000000000", "0x1000000000000000000000000000000000000000")
	oracle.AddSubscriptionIfNotAlready(uint64(6434), "0x2000000000000000000000000000000000000000", "0x2000000000000000000000000000000000000000")
	oracle.AddSubscriptionIfNotAlready(uint64(643344), "0x2000000000000000000000000000000000000000", "0x2000000000000000000000000000000000000000")

	oracle.StoreLatestOnchainState()

	subs := []*contract.ContractSubscribeValidator{
		&contract.ContractSubscribeValidator{
			ValidatorID:            33,
			SubscriptionCollateral: big.NewInt(1000),
			Raw:                    types.Log{TxHash: [32]byte{0x1}, Topics: []common.Hash{{0x2}}},
			Sender:                 common.Address{148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}
	oracle.state.Subscriptions = subs
	_ = subs

	defer os.Remove(filepath.Join(StateFileName, StateFolder))
	defer os.RemoveAll(StateFolder)
	oracle.SaveToJson()

	oracle.LoadFromJson()
	//defer os.Remove(filepath.Join(StateFileName, StateFolder))
	//defer os.RemoveAll(StateFolder)
	jsonData, err := json.MarshalIndent(oracle.state, "", " ")
	if err != nil {
		log.Fatal("could not marshal state to json: ", err)
	}

	fmt.Printf("recovered data: %s\n", jsonData)
	log.Info(oracle.state.Validators[3].ValidatorStatus)

	require.Equal(t, oracle.state, oracle.state)

	//require.NoError(t, err)
	//require.Equal(t, state, state)
}

func Test_SaveLoadFromToFile_EmptyState(t *testing.T) {
	oracle := NewOracle(&Config{
		PoolAddress:     "0x0000000000000000000000000000000000000000",
		PoolFeesAddress: "0x1000000000000000000000000000000000000000",
		Network:         "mainnet",
	})

	oracle.SaveStateToFile()
	defer os.Remove(filepath.Join(StateFileName, StateFolder))
	defer os.RemoveAll(StateFolder)

	err := oracle.LoadStateFromFile()
	require.NoError(t, err)
	require.Equal(t, oracle.state, oracle.state)
}
func Test_SaveLoadFromToFile_PopulatedState(t *testing.T) {

	oracle := NewOracle(&Config{
		PoolAddress:     "0x0000000000000000000000000000000000000000",
		PoolFeesAddress: "0x1000000000000000000000000000000000000000",
		Network:         "mainnet",
	})

	oracle.state.Donations = make([]Donation, 1)

	oracle.state.Donations[0] = Donation{
		AmountWei: big.NewInt(1000),
		Block:     1000,
		TxHash:    "0x",
	}

	oracle.state.Validators[10] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(1000),
		PendingRewardsWei:     big.NewInt(1000),
		CollateralWei:         big.NewInt(1000),
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        10,
		ValidatorKey:          "0xc", // TODO: Fix this, should be uint64
	}

	oracle.state.Validators[20] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(13000),
		PendingRewardsWei:     big.NewInt(100),
		CollateralWei:         big.NewInt(1000000),
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        20,
		ValidatorKey:          "0xc",
	}

	oracle.state.Validators[30] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(53000),
		PendingRewardsWei:     big.NewInt(000),
		CollateralWei:         big.NewInt(4000000),
		WithdrawalAddress:     "0xa000000000000000000000000000000000000000",
		ValidatorIndex:        30,
		ValidatorKey:          "0xc",
	}

	defer os.Remove(filepath.Join(StateFileName, StateFolder))
	defer os.RemoveAll(StateFolder)
	oracle.SaveStateToFile()

	err := oracle.LoadStateFromFile()
	require.NoError(t, err)
	require.Equal(t, oracle.state, oracle.state)
}

func Test_IsValidatorSubscribed(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.state.Validators[10] = &ValidatorInfo{
		ValidatorStatus:       Active,
		AccumulatedRewardsWei: big.NewInt(100),
		PendingRewardsWei:     big.NewInt(200),
	}
	oracle.state.Validators[20] = &ValidatorInfo{
		ValidatorStatus:       YellowCard,
		AccumulatedRewardsWei: big.NewInt(300),
		PendingRewardsWei:     big.NewInt(300),
	}
	oracle.state.Validators[30] = &ValidatorInfo{
		ValidatorStatus:       RedCard,
		AccumulatedRewardsWei: big.NewInt(900),
		PendingRewardsWei:     big.NewInt(100),
	}
	oracle.state.Validators[40] = &ValidatorInfo{
		ValidatorStatus:       NotSubscribed,
		AccumulatedRewardsWei: big.NewInt(50),
		PendingRewardsWei:     big.NewInt(10),
	}
	oracle.state.Validators[40] = &ValidatorInfo{
		ValidatorStatus:       Banned,
		AccumulatedRewardsWei: big.NewInt(50),
		PendingRewardsWei:     big.NewInt(10),
	}
	require.Equal(t, true, oracle.isSubscribed(10))
	require.Equal(t, true, oracle.isSubscribed(20))
	require.Equal(t, true, oracle.isSubscribed(30))
	require.Equal(t, false, oracle.isSubscribed(40))
	require.Equal(t, false, oracle.isSubscribed(50))
}

func Test_BanValidator(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.AddSubscriptionIfNotAlready(1, "0xa", "0xb")
	oracle.AddSubscriptionIfNotAlready(2, "0xa", "0xb")
	oracle.AddSubscriptionIfNotAlready(3, "0xa", "0xb")

	// New reward arrives
	oracle.IncreaseAllPendingRewards(big.NewInt(99))

	// Shared equally among all validators
	require.Equal(t, big.NewInt(33), oracle.state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(33), oracle.state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(33), oracle.state.Validators[3].PendingRewardsWei)

	// Ban validator 3
	oracle.handleBanValidator(Block{ValidatorIndex: 3})

	// Its pending balance is shared equally among the rest
	require.Equal(t, big.NewInt(49), oracle.state.Validators[1].PendingRewardsWei)
	require.Equal(t, big.NewInt(49), oracle.state.Validators[2].PendingRewardsWei)
	require.Equal(t, big.NewInt(0), oracle.state.Validators[3].PendingRewardsWei)

	// The pool fee address gets the rounding errors (1 wei, neglectable)
	require.Equal(t, big.NewInt(1), oracle.state.PoolAccumulatedFees)
}

func Test_IsBanned(t *testing.T) {
	oracle := NewOracle(&Config{})
	oracle.state.Validators[1] = &ValidatorInfo{
		ValidatorStatus: Active,
	}
	oracle.state.Validators[2] = &ValidatorInfo{
		ValidatorStatus: YellowCard,
	}
	oracle.state.Validators[3] = &ValidatorInfo{
		ValidatorStatus: RedCard,
	}
	oracle.state.Validators[4] = &ValidatorInfo{
		ValidatorStatus: NotSubscribed,
	}
	oracle.state.Validators[5] = &ValidatorInfo{
		ValidatorStatus: Banned,
	}

	require.Equal(t, false, oracle.IsBanned(1))
	require.Equal(t, false, oracle.IsBanned(2))
	require.Equal(t, false, oracle.IsBanned(3))
	require.Equal(t, false, oracle.IsBanned(4))
	require.Equal(t, true, oracle.IsBanned(5))
}

// TODO: Add a Test_Handle_Subscriptions_1 happy path to cover the normal flow

// Follows an non happy path with a lot of edge cases and possible misconfigurations
func Test_Handle_TODO(t *testing.T) {
	/*
		cfg := &Config{
			PoolFeesAddress: "0xa",
			PoolFeesPercent: 0,
			CollateralInWei: big.NewInt(1000000),
		}

			state := NewOracleState(cfg)

			// Two subscriptions ok. Third not enough collateral
			subs := []Subscription{
				{
					ValidatorIndex: 10,
					ValidatorKey:   "0xaa",
					Collateral:     big.NewInt(1000000), // Enough
					BlockNumber:    0,
					TxHash:         "0xab",
					WithdrawalAddress: "0xac",
				},
				{
					ValidatorIndex: 20,
					ValidatorKey:   "0xba",
					Collateral:     big.NewInt(1000000), // Enough
					BlockNumber:    0,
					TxHash:         "0xbb",
					WithdrawalAddress: "0xbc",
				},
				{
					ValidatorIndex: 30,
					ValidatorKey:   "0xba",
					Collateral:     big.NewInt(50), // Not enough
					BlockNumber:    0,
					TxHash:         "0xbb",
					WithdrawalAddress: "0xbc",
				},
			}
			oracle.handleManualSubscriptions(cfg.CollateralInWei, subs)

			// Block from a subscribed validator (manual)
			block1 := Block{
				Slot:           0,
				ValidatorIndex: 10,
				ValidatorKey:   "0x",
				Reward:         big.NewInt(50000000),
				RewardType:     VanilaBlock,
				WithdrawalAddress: "0ac",
			}
			state.handleCorrectBlockProposal(block1)

			// Block from a non-subscribed validator (auto)
			block2 := Block{
				Slot:           0,
				ValidatorIndex: 40,
				ValidatorKey:   "0x",
				Reward:         big.NewInt(3333333),
				RewardType:     VanilaBlock,
				WithdrawalAddress: "0ac",
			}
			state.handleCorrectBlockProposal(block2)

			fmt.Println(oracle.state.Validators[10])
			fmt.Println(oracle.state.Validators[20])
			fmt.Println(oracle.state.Validators[30])

			// Test also
			//or.State.handleBanValidator(customBlock)
			//or.oracle.handleManualUnsubscriptions(newBlockUnsub)
			//or.oracle.handleDonations(blockDonations)
			//or.State.handleMissedBlock(customBlock)
	*/
}

// TODO: Add tests for add subscription and remove subscription
// TODO: Add more tests when spec settled

func Test_CanValidatorSubscribeToPool(t *testing.T) {

	require.Equal(t, CanValidatorSubscribeToPool(&v1.Validator{
		Status: v1.ValidatorStatePendingInitialized,
	}), true)

	require.Equal(t, CanValidatorSubscribeToPool(&v1.Validator{
		Status: v1.ValidatorStatePendingQueued,
	}), true)

	require.Equal(t, CanValidatorSubscribeToPool(&v1.Validator{
		Status: v1.ValidatorStateActiveOngoing,
	}), true)
}

// Test to estimate how much memory the validator state will take with 2000 validators,
// each one proposing a block
func Test_ValidatorInfoSize(t *testing.T) {
	for i := 0; i < 3; i++ {
		oracle := NewOracle(&Config{
			CollateralInWei: big.NewInt(1000),
		})

		oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{
			33: &v1.Validator{
				Index:  33,
				Status: v1.ValidatorStateActiveOngoing,
				Validator: &phase0.Validator{
					// Valid eth1 address: 0x9427a30991170f917d7b83def6e44d26577871ed
					WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					// Valdator pubkey: 0x81aae709e6aee7ed49cd15b941d85b967afcc8b844ee20bc7e13962e8484572c1b43d4be75652119ec353c1a32443e0d
					PublicKey: phase0.BLSPubKey{129, 170, 231, 9, 230, 174, 231, 237, 73, 205, 21, 185, 65, 216, 91, 150, 122, 252, 200, 184, 68, 238, 32, 188, 126, 19, 150, 46, 132, 132, 87, 44, 27, 67, 212, 190, 117, 101, 33, 25, 236, 53, 60, 26, 50, 68, 62, 13},
				},
			},
		}

		//save state of 2000 validators
		numValidators := 2000

		//create 2000 validators with index 0-1999
		valsID := make([]uint64, numValidators)
		for i := 0; i < numValidators; i++ {
			valsID[i] = uint64(i)
		}
		//subscribe 2000 validators
		subs := new_subs_slice(common.HexToAddress("0x0123456789abcdef0123456789abcdef01234567"), valsID, big.NewInt(1000))
		oracle.handleManualSubscriptions(subs)

		//make 2000 validators propose a block
		for i := 0; i < numValidators; i++ {
			oracle.handleCorrectBlockProposal(Block{
				Slot:              uint64(i),
				ValidatorIndex:    uint64(valsID[0]),
				ValidatorKey:      "0x0123456789abcdef0123456789abcdef01234567",
				Reward:            big.NewInt(5000000000000000000), // 0.5 eth of reward
				RewardType:        MevBlock,
				WithdrawalAddress: "0x0123456789abcdef0123456789abcdef01234567",
			})
		}

		// //make 2000 validators miss a block
		// for i := 0; i < numValidators; i++ {
		// 	state.handleMissedBlock(Block{
		// 		Slot:              uint64(100),
		// 		ValidatorIndex:    uint64(valsID[i]),
		// 		ValidatorKey:      "0x0123456789abcdef0123456789abcdef01234567",
		// 		Reward:            big.NewInt(5000000000000000000),
		// 		RewardType:        VanilaBlock,
		// 		WithdrawalAddress: "0x0123456789abcdef0123456789abcdef01234567",
		// 	})
		// }

		// //make 2000 validators propose a block
		// for i := 0; i < numValidators; i++ {
		// 	state.handleCorrectBlockProposal(Block{
		// 		Slot:              uint64(100),
		// 		ValidatorIndex:    uint64(valsID[i]),
		// 		ValidatorKey:      "0x0123456789abcdef0123456789abcdef01234567",
		// 		Reward:            big.NewInt(5000000000000000000), // 0.5 eth of reward
		// 		RewardType:        MevBlock,
		// 		WithdrawalAddress: "0x0123456789abcdef0123456789abcdef01234567",
		// 	})
		// }
		oracle.SaveStateToFile()
		filePath := "oracle-data/state.gob"

		// Get file information
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Fatal(err)
		}

		// Get file size in bytes
		fileSize := fileInfo.Size()
		fileSizeMB := float64(fileSize) / (1024 * 1024)

		// Print the file size
		log.Info("File size:", fileSizeMB, "MB")
	}
}

// This test tries to mock a real time scenario where 2000 validators are tracked by the pool,
// and tries check how much memory the oracleState takes.
// In this scenario, the oracle uploads a new state to the chain once every 3 days.
// Since we have 2000 validators, each time the state is uploaded to the chain,
// a rough estimate of 100 blocks will have been proposed by the validators.
func Test_SizeMultipleOnchainState(t *testing.T) {

	oracle := NewOracle(&Config{
		CollateralInWei: big.NewInt(1000),
		PoolFeesAddress: "0x1123456789abcdef0123456789abcdef01234568",
	})

	oracle.beaconValidators = map[phase0.ValidatorIndex]*v1.Validator{}

	//prepare 2000 validators
	numValidators := 2000

	//create 2000 validators with index 0-1999
	valsID := make([]uint64, numValidators)
	for i := 0; i < numValidators; i++ {
		valsID[i] = uint64(i)
	}

	for i := 0; i < len(valsID); i++ {
		address := common.HexToAddress("0x0123456789abcdef0123456789abcdef01234567")
		oracle.beaconValidators[phase0.ValidatorIndex(i)] = &v1.Validator{
			Index:  phase0.ValidatorIndex(valsID[i]),
			Status: v1.ValidatorStateActiveOngoing,
			Validator: &phase0.Validator{
				// withdrawal credentials = 0x(valID)0000..000
				WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, address[0], address[1], address[2], address[3], address[4], address[5], address[6], address[7], address[8], address[9], address[10], address[11], address[12], address[13], address[14], address[15], address[16], address[17], address[18], address[19]},
				// Valdator pubkey: 0x(valID)0000...000
				PublicKey: phase0.BLSPubKey{byte(valsID[i]), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
		}
	}
	//subscribe 2000 validators. All validators will have the same withdrawal address.
	subs := new_subs_slice(common.HexToAddress("0x0123456789abcdef0123456789abcdef01234567"), valsID, big.NewInt(1000))

	//oracle handles the subscriptions.
	oracle.handleManualSubscriptions(subs)

	//simulate the scenario. In one year, we will commit 121 states to the chain.
	//each time the state is commited, a rough estimate of 100 blocks will have been proposed by the validators.
	for i := 0; i < 121; i++ {
		for j := 0; j < 100; j++ {
			oracle.handleCorrectBlockProposal(Block{
				Slot:              uint64(100),
				ValidatorIndex:    uint64(valsID[j]),
				ValidatorKey:      "0x0123456789abcdef0123456789abcdef01234567",
				Reward:            big.NewInt(5000000000000000000), // 0.5 eth of reward
				RewardType:        MevBlock,
				WithdrawalAddress: "0x0100000000000000000000009b3b13d6b6f3f52154a8b00d818392b61e4b42b4",
			})
		}
		// the "StoreLatestOnchainState" function is responsible of making a deep copy of all
		// current validator data and storing it in the new "state.CommitedStates" map, which
		// contains all the past onchain states of the validators.
		// each time we store a new latestOnchainState, the merkleroot has changed, so we
		// store a new state of all the validators.
		// in a year, will update the onchain state 121 times. each time we do this, we will
		// store the last onchain state, which contains the information of all the validators
		oracle.StoreLatestOnchainState()
	}

	//after 1 year, we will have 121 states in the "state.CommitedStates" map.
	//each state contains the information of 2000 validators.

	//save the state to a file
	oracle.SaveStateToFile()
	filePath := "oracle-data/state.gob"

	// Get file information
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}

	// Get file size in bytes
	fileSize := fileInfo.Size()
	fileSizeMB := float64(fileSize) / (1024 * 1024)

	// Print the file size
	log.Info("File size:", fileSizeMB, "MB")
}

// returns len(valsID) new valid subscriptions
func new_subs_slice(address common.Address, valsID []uint64, collateral *big.Int) []*contract.ContractSubscribeValidator {
	subs := make([]*contract.ContractSubscribeValidator, len(valsID))
	for i := 0; i < len(valsID); i++ {
		subs[i] = &contract.ContractSubscribeValidator{
			ValidatorID:            valsID[i],
			SubscriptionCollateral: collateral,
			Raw:                    types.Log{TxHash: [32]byte{0x1}},
			Sender:                 address,
		}
	}
	return subs
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

func blockOkProposal(slot uint64, valIndex uint64, pubKey string, reward *big.Int, withAddress string) Block {
	return Block{
		Slot:              slot,
		ValidatorIndex:    valIndex,
		ValidatorKey:      pubKey,
		BlockType:         OkPoolProposal,
		Reward:            reward,
		RewardType:        MevBlock,
		WithdrawalAddress: withAddress,
	}
}
