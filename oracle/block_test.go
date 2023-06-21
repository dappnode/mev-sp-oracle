package oracle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/dappnode/mev-sp-oracle/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/stretchr/testify/require"
)

func Test_Getters_Bellatrix(t *testing.T) {

	block := &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionBellatrix,
		Bellatrix: &bellatrix.SignedBeaconBlock{
			Message: &bellatrix.BeaconBlock{
				Slot:          5214140,
				ProposerIndex: 12,
				Body: &bellatrix.BeaconBlockBody{
					ExecutionPayload: &bellatrix.ExecutionPayload{
						FeeRecipient: [20]byte{56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151},
						BlockNumber:  1000,
						Transactions: []bellatrix.Transaction{
							{1, 2},
							{1, 2}},
					},
				},
			},
		}}

	fullBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5214140,
		ValidatorIndex: phase0.ValidatorIndex(12)},
		&v1.Validator{
			Index: 12,
		})
	fullBlock.SetConsensusBlock(block)

	require.Equal(t, [32]uint8([32]uint8{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}), fullBlock.GetBaseFeePerGas())
	require.Equal(t, uint64(0), fullBlock.GetGasUsed())
	require.Equal(t, uint64(12), fullBlock.GetProposerIndexUint64())
	require.Equal(t, phase0.ValidatorIndex(12), fullBlock.GetProposerIndex())
	require.Equal(t, uint64(5214140), fullBlock.GetSlotUint64())
	require.Equal(t, phase0.Slot(5214140), fullBlock.GetSlot())
	require.Equal(t, big.NewInt(1000), fullBlock.GetBlockNumberBigInt())
	require.Equal(t, uint64(1000), fullBlock.GetBlockNumber())
	require.Equal(t, []bellatrix.Transaction([]bellatrix.Transaction{{0x1, 0x2}, {0x1, 0x2}}), fullBlock.GetBlockTransactions())
	require.Equal(t, "0x388C818CA8B9251b393131C08a736A67ccB19297", fullBlock.GetFeeRecipient())
}

func Test_Getters_Capella(t *testing.T) {

	block := &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionCapella,
		Capella: &capella.SignedBeaconBlock{
			Message: &capella.BeaconBlock{
				Slot:          5214140,
				ProposerIndex: 12,
				Body: &capella.BeaconBlockBody{
					ExecutionPayload: &capella.ExecutionPayload{
						FeeRecipient: [20]byte{56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151},
						BlockNumber:  1000,
						Transactions: []bellatrix.Transaction{
							{5, 6},
							{7, 8}},
					},
				},
			},
		}}

	fullBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5214140,
		ValidatorIndex: phase0.ValidatorIndex(12)},
		&v1.Validator{
			Index: 12,
		})
	fullBlock.SetConsensusBlock(block)

	require.Equal(t, [32]uint8([32]uint8{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}), fullBlock.GetBaseFeePerGas())
	require.Equal(t, uint64(0), fullBlock.GetGasUsed())
	require.Equal(t, uint64(12), fullBlock.GetProposerIndexUint64())
	require.Equal(t, phase0.ValidatorIndex(12), fullBlock.GetProposerIndex())
	require.Equal(t, uint64(5214140), fullBlock.GetSlotUint64())
	require.Equal(t, phase0.Slot(5214140), fullBlock.GetSlot())
	require.Equal(t, big.NewInt(1000), fullBlock.GetBlockNumberBigInt())
	require.Equal(t, uint64(1000), fullBlock.GetBlockNumber())
	require.Equal(t, []bellatrix.Transaction([]bellatrix.Transaction{{0x5, 0x6}, {0x7, 0x8}}), fullBlock.GetBlockTransactions())
	require.Equal(t, "0x388C818CA8B9251b393131C08a736A67ccB19297", fullBlock.GetFeeRecipient())
}

// This test uses real mocked blocks that can be fetched and stores with this util:
// Test_GetFullBlockAtSlot (see onchain_test.go)
func Test_FullBlock_All(t *testing.T) {

	// Run locally. Disabled since in CI we have some issues with git lfs bandwidth free limits
	t.Skip("Skipping test")

	type donation struct {
		Hash   string
		Amount *big.Int
	}

	type test struct {
		// Input
		Name         string
		Slot         uint64
		WithHeeaders bool
		ChainId      string
		PoolAddress  string

		// Output
		ExpectedTip             *big.Int
		ExpectedMevReward       *big.Int
		ExpectedProposedIndex   phase0.ValidatorIndex
		ExpectedDonations       []*donation
		ExpectedHasMev          bool
		ExpectedRewardSent      bool
		ExpectedRewardType      RewardType
		ExpectedReward          *big.Int
		ExpectedFeeRecipient    string
		ExpectedMEVFeeRecipient string
	}

	tests := []test{
		// prater.beaconcha.in/slot/5214302: vanila block, no mev, no donation, was not sent to the pool
		{ /*in->*/ "1", uint64(5214302), true, "5", "0xF21fbbA423f3a893A2402d68240B219308AbCA46" /*expected->*/, big.NewInt(38657065851824731), big.NewInt(0), phase0.ValidatorIndex(218475), []*donation{}, false, false, VanilaBlock, big.NewInt(38657065851824731), "0x4D496CcC28058B1D74B7a19541663E21154f9c84", ""},

		// prater.beaconcha.in/slot/5214321: mev reward, no donations, was not sent to the pool
		{ /*in->*/ "2", uint64(5214321), true, "5", "0xF21fbbA423f3a893A2402d68240B219308AbCA46" /*expected->*/, big.NewInt(15992505660349526), big.NewInt(15867629069461526), phase0.ValidatorIndex(252922), []*donation{}, true, false, MevBlock, big.NewInt(15867629069461526), "0x8dC847Af872947Ac18d5d63fA646EB65d4D99560", "0x4d496ccc28058b1d74b7a19541663e21154f9c84"},

		// prater.beaconcha.in/slot/5307527: vanila block, no donations, reward sent to pool
		{ /*in->*/ "3", uint64(5307527), true, "5", "0x000095E79eAC4d76aab57cB2c1f091d553b36ca0" /*expected->*/, big.NewInt(105735750887810922), big.NewInt(0), phase0.ValidatorIndex(289213), []*donation{}, false, true, VanilaBlock, big.NewInt(105735750887810922), "0x000095E79eAC4d76aab57cB2c1f091d553b36ca0", ""},

		// prater.beaconcha.in/slot/5320337: vanila block, no donations, not sent to pool
		{ /*in->*/ "4", uint64(5320337), true, "5", "0xF21fbbA423f3a893A2402d68240B219308AbCA46" /*expected->*/, big.NewInt(41380243736782800), big.NewInt(0), phase0.ValidatorIndex(32553), []*donation{}, false, false, VanilaBlock, big.NewInt(41380243736782800), "0x4D496CcC28058B1D74B7a19541663E21154f9c84", ""},

		// prater.beaconcha.in/slot/5320342: vanila block, no donations, sent to pool
		{ /*in->*/ "5", uint64(5320342), true, "5", "0xc6e2459991BfE27cca6d86722F35da23A1E4Cb97" /*expected->*/, big.NewInt(117335955724211704), big.NewInt(0), phase0.ValidatorIndex(102472), []*donation{}, false, true, VanilaBlock, big.NewInt(117335955724211704), "0xc6e2459991BfE27cca6d86722F35da23A1E4Cb97", ""},

		// prater.beaconcha.in/slot/5344344: vanila block, no donations, not sent to pool
		{ /*in->*/ "6", uint64(5344344), true, "5", "0xF21fbbA423f3a893A2402d68240B219308AbCA46" /*expected->*/, big.NewInt(54473697141591874), big.NewInt(0), phase0.ValidatorIndex(224284), []*donation{}, false, false, VanilaBlock, big.NewInt(54473697141591874), "0x4D496CcC28058B1D74B7a19541663E21154f9c84", ""},

		// prater.beaconcha.in/slot/5862054: mev block, contains donation to pool, mev not sent to pool (the donation is a normal tx)
		{ /*in->*/ "7", uint64(5862054), true, "5", "0xF21fbbA423f3a893A2402d68240B219308AbCA46" /*expected->*/, big.NewInt(76416355735251731), big.NewInt(76416210831135731), phase0.ValidatorIndex(230624), []*donation{{"0xb647b7c050625d565d8466db954fb6ee2976135ee12b82b25ac308e93fe3e1f4", big.NewInt(113500000000000000)}}, true, false, MevBlock, big.NewInt(76416210831135731), "0xfC0157aA4F5DB7177830ACddB3D5a9BB5BE9cc5e", "0x388ea662ef2c223ec0b047d41bf3c0f362142ad5"},

		// prater.beaconcha.in/slot/5862104: vanila block, contains donation to pool, reward did not go to pool (the donation is done via a smart contract tx aka internal)
		{ /*in->*/ "8", uint64(5862104), true, "5", "0xF21fbbA423f3a893A2402d68240B219308AbCA46" /*expected->*/, big.NewInt(3110023195815608), big.NewInt(0), phase0.ValidatorIndex(423933), []*donation{{"0x7446efc78c4e6bdc17d5c2266ea43415edf4aee5fb439883136f8eeeffc2f6fe", big.NewInt(43234345)}}, false, false, VanilaBlock, big.NewInt(3110023195815608), "0x94750381bE1AbA0504C666ee1DB118F68f0780D4", ""},

		// https://prater.beaconcha.in/slot/5864096: mev block, no donations, mev reward was sent to pool
		{ /*in->*/ "9", uint64(5864096), true, "5", "0xf21fbba423f3a893a2402d68240b219308abca46" /*expected->*/, big.NewInt(5759373075631516), big.NewInt(5759373075365746), phase0.ValidatorIndex(408154), []*donation{}, true, true, MevBlock, big.NewInt(5759373075365746), "0x8dC847Af872947Ac18d5d63fA646EB65d4D99560", "0xf21fbba423f3a893a2402d68240b219308abca46"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			fullBlock, err := LoadFullBlock(tt.Slot, tt.ChainId, tt.WithHeeaders)
			require.NoError(t, err)

			// Calculate parameters
			proposerTip, err := fullBlock.GetProposerTip()
			require.NoError(t, err)
			feeRecipient := fullBlock.GetFeeRecipient()
			proposerIndex := fullBlock.GetProposerIndex()
			donations := fullBlock.GetDonations(tt.PoolAddress)
			sentReward, sent, rewardType := fullBlock.GetSentRewardAndType(tt.PoolAddress, tt.WithHeeaders)
			mevReward, mevFound, mevRecipient := fullBlock.MevRewardInWei()

			// Assert
			require.Equal(t, tt.ExpectedFeeRecipient, feeRecipient)
			require.Equal(t, tt.ExpectedProposedIndex, proposerIndex)
			require.Equal(t, len(tt.ExpectedDonations), len(donations))
			for i := 0; i < len(tt.ExpectedDonations); i++ {
				require.Equal(t, tt.ExpectedDonations[i].Hash, donations[i].Raw.TxHash.String())
				require.Equal(t, tt.ExpectedDonations[i].Amount, donations[i].DonationAmount)

			}
			require.Equal(t, tt.ExpectedTip, proposerTip)
			require.Equal(t, tt.ExpectedRewardSent, sent)
			require.Equal(t, tt.ExpectedRewardType, rewardType)
			require.Equal(t, tt.ExpectedReward, sentReward)
			require.Equal(t, tt.ExpectedMEVFeeRecipient, mevRecipient)
			require.Equal(t, tt.ExpectedMevReward, mevReward)
			require.Equal(t, tt.ExpectedHasMev, mevFound)
		})
	}

}

func Test_SummarizedBlock(t *testing.T) {

	// Run locally. Disabled since in CI we have some issues with git lfs bandwidth free limits
	t.Skip("Skipping test")

	type test struct {
		// Input
		Name               string
		PoolAddress        string
		ProposerSubscribed bool
		Slot               uint64

		// Output
		ExpectedBlock          uint64
		ExpectedBlockType      BlockType
		ExpectedRewardType     RewardType
		ExpectedReward         *big.Int
		ExpectedValidatorIndex uint64
		ExpectedValKey         string
		ExpeectedWithCred      string
	}

	tests := []test{
		// subscribed validator proposes mev block with correct fee https://prater.beaconcha.in/slot/5739624
		{"1", "0xf4e8263979a89dc357d7f9f79533febc7f3e287b", true, uint64(5739624), uint64(9086632), OkPoolProposal, MevBlock, big.NewInt(23547931077241917), uint64(234515), "0xa2240e4a358a4f87dfece4c85f08b41abda91b558fe2e544885ed21163681576f41af2ec0161955c735803adb5fee910", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},

		// subscribed validator proposes vanila block with correct fee https://prater.beaconcha.in/slot/5739629
		{"2", "0x94750381be1aba0504c666ee1db118f68f0780d4", true, uint64(5739629), uint64(9086637), OkPoolProposal, VanilaBlock, big.NewInt(15960095948338108), uint64(426736), "0xb6283b7cc2eaedde6f0ced4bffb8bc99c1e9cb3de77d6be8be02bf78fa850b74ee57f6b960fc48ca0ccd4b683521f3f9", "0x59b0d71688da01057c08e4c1baa8faa629819c2a"},

		// non subscribed validator proposes vanila block with correct fee https://prater.beaconcha.in/slot/5739634
		{"3", "0xa111B576408B1CcDacA3eF26f22f082C49bcaa55", false, uint64(5739634), uint64(9086639), OkPoolProposal, VanilaBlock, big.NewInt(41035389197072885), uint64(408206), "0xa57f9cbd211d3219ac54c8f329d1e2a4c65c54978444d7e5ff71d6129dd33ebc2e26bdfd611fc391a7a84b4d43418ac0", "0xa111b576408b1ccdaca3ef26f22f082c49bcaa55"},

		// non subscribed validator proposes mev block with correct fee https://prater.beaconcha.in/slot/5739644
		{"4", "0xF4e8263979A89Dc357d7f9F79533Febc7f3e287B", false, uint64(5739644), uint64(9086648), OkPoolProposal, MevBlock, big.NewInt(37799556930427516), uint64(234604), "0xb67e026940ccc26a478dcb020767d1391ccd6dc1f66f5bee328750cbbc4eb909665f7340c58411b6c29c01bdca3951c4", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},

		// subscribed validator proposes a mev block with wrong fee recipient https://prater.beaconcha.in/slot/5739624
		{"5", "0x0000000000000000000000000000000000000000", true, uint64(5739624), uint64(9086632), WrongFeeRecipient, MevBlock, big.NewInt(23547931077241917), uint64(234515), "0xa2240e4a358a4f87dfece4c85f08b41abda91b558fe2e544885ed21163681576f41af2ec0161955c735803adb5fee910", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},

		// subscribed validator proposes a vanila block with wrong fee recipient https://prater.beaconcha.in/slot/5739637
		{"6", "0x0000000000000000000000000000000000000000", true, uint64(5739637), uint64(9086642), WrongFeeRecipient, VanilaBlock, big.NewInt(11591726353544658), uint64(468452), "0x8371d199579f91a966732bf5eaaa940ac037084f95018ddd6530f9003c6b028f0181f52b50bdbe692f49f72c6fc9ad38", "0x0158fea37a1654d872c19f8326df00b7cb07c5cf"},

		// non subscribed validator proposes a block with wrong fee recipient (kind of ignored) https://prater.beaconcha.in/slot/5739637
		{"7", "0x0000000000000000000000000000000000000000", false, uint64(5739637), uint64(9086642), WrongFeeRecipient, UnknownRewardType, big.NewInt(0), uint64(468452), "0x8371d199579f91a966732bf5eaaa940ac037084f95018ddd6530f9003c6b028f0181f52b50bdbe692f49f72c6fc9ad38", "0x0158fea37a1654d872c19f8326df00b7cb07c5cf"},

		// subscribed validator misses a block https://prater.beaconcha.in/slot/5739640
		{"8", "0x0000000000000000000000000000000000000000", true, uint64(5739640), uint64(0), MissedProposal, UnknownRewardType, big.NewInt(0), uint64(458817), "0xb3fda21f2e4d6d93432d0d70c83c81159b2c625576eadbab80a2b55538ebd54a975cdc8a5cbb3909bbbb02bd08a3a009", "0x0997fdeffd9d29710436b2155ed702d845f7061a"},

		// unsubscribed validator misses a block (kind of ignored) https://prater.beaconcha.in/slot/5739640
		{"9", "0x0000000000000000000000000000000000000000", false, uint64(5739640), uint64(0), MissedProposal, UnknownRewardType, big.NewInt(0), uint64(458817), "0xb3fda21f2e4d6d93432d0d70c83c81159b2c625576eadbab80a2b55538ebd54a975cdc8a5cbb3909bbbb02bd08a3a009", "0x0997fdeffd9d29710436b2155ed702d845f7061a"},

		// subscribed validator proposes a block with correct fee recipient but BLS credentials (note: this test can fail if withdrawal is updated) https://prater.beaconcha.in/slot/5739736
		{"10", "0xe0a2Bd4258D2768837BAa26A28fE71Dc079f84c7", true, uint64(5739736), uint64(9086730), OkPoolProposalBlsKeys, VanilaBlock, big.NewInt(12805869897561244), uint64(319479), "0xb3e1c989c0d27824da29480a4bc09f4c561c2ce75d0a2ba7b3a57480d93d5ddb627d5fa0923402fd33145ded5eaa9d98", "0x95068c3ce9e71d7d4ca51df4230045e150d28d6c49727cb0d994d50b1cdeff"},

		// non subscribed validator proposes a vanila block with a wrong fee recipient (kind of ignored) most blocks are this https://prater.beaconcha.in/slot/5739707
		// reward is not calculated as its very expensive
		{"11", "0x0000000000000000000000000000000000000000", false, uint64(5739707), uint64(9086704), WrongFeeRecipient, UnknownRewardType, big.NewInt(0), uint64(474819), "0xa20fb16d127a22c7502e70db4eef33d1f11070d8bb232c91bf2b8beeadae8836d02774f7b5e96893ed80e9c7020e0d2a", "0x5bdd7b7a48d146b23969218eac5f152760bc072e"},

		// non subscribed validator proposes a mev block with a wrong fee recipient (kind of ignored) most blocks are this https://prater.beaconcha.in/slot/5739722
		// reward is calculated. not used but cheap to calculate it
		{"12", "0x0000000000000000000000000000000000000000", false, uint64(5739722), uint64(9086717), WrongFeeRecipient, MevBlock, big.NewInt(28327464143130026), uint64(232204), "0xb1294f2c149ee1cd0b2d9dd8bd8781cb4920353623426e64eb4a915b553c4dbefea53bc8c83f6b3dcee44223bdcd3c6c", "0x8f0844fd51e31ff6bf5babe21dccf7328e19fd9f"},

		// missed block
		{"13", "0x0000000000000000000000000000000000000000", true, uint64(5320341), uint64(0), MissedProposal, UnknownRewardType, big.NewInt(0), uint64(179637), "0x8bd7e3f2896b0cdeb42dc25053086ddc2fda0afcc1a4e6b1f7a048d18d0445f71d46db067318ee0238ca6ec705e471d4", "0x653db96d58d6cce73be5e565d907d8c45bc8bff6a0f04f1a21498671ab204a"},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			oracle := NewOracle(&Config{})
			oracle.state.PoolAddress = tt.PoolAddress

			if tt.ProposerSubscribed {
				oracle.addSubscription(tt.ExpectedValidatorIndex, "0x", "0x")
			}

			fullBlock, err := LoadFullBlock(tt.Slot, "5", tt.ProposerSubscribed)
			require.NoError(t, err)
			block := fullBlock.SummarizedBlock(oracle, tt.PoolAddress)

			require.Equal(t, tt.Slot, block.Slot)
			require.Equal(t, tt.ExpectedBlock, block.Block)
			require.Equal(t, tt.ExpectedBlockType, block.BlockType)
			require.Equal(t, tt.ExpectedRewardType, block.RewardType)
			require.Equal(t, tt.ExpectedReward, block.Reward)
			require.Equal(t, tt.ExpectedValidatorIndex, block.ValidatorIndex)
			require.Equal(t, tt.ExpectedValKey, block.ValidatorKey)
			require.Equal(t, tt.ExpeectedWithCred, block.WithdrawalAddress)
		})
	}
}

func Test_Marashal_FullBlock(t *testing.T) {

	// Creates some test data to mock
	proposalDuty := &v1.ProposerDuty{
		Slot:           5214140,
		ValidatorIndex: phase0.ValidatorIndex(12)}

	validator := &v1.Validator{
		Index: 12,
		Validator: &phase0.Validator{
			PublicKey:             phase0.BLSPubKey{1, 2, 3},
			WithdrawalCredentials: []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
		},
	}

	block := &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionBellatrix,
		Bellatrix: &bellatrix.SignedBeaconBlock{
			Message: &bellatrix.BeaconBlock{
				Slot:          5214140,
				ProposerIndex: 12,
				Body: &bellatrix.BeaconBlockBody{
					ExecutionPayload: &bellatrix.ExecutionPayload{
						FeeRecipient: [20]byte{56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151},
						BlockNumber:  8745218,
						Transactions: []bellatrix.Transaction{
							{1, 2},
							{1, 2}},
					},
					ETH1Data: &phase0.ETH1Data{
						DepositRoot: phase0.Root{},
						BlockHash:   []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 39, 163, 9, 145, 23, 15, 145, 125, 123, 131, 222, 246, 228, 77, 38, 87, 120, 113, 237},
					},
					ProposerSlashings: []*phase0.ProposerSlashing{},
					AttesterSlashings: []*phase0.AttesterSlashing{},
					Attestations:      []*phase0.Attestation{},
					Deposits:          []*phase0.Deposit{},
					VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
					SyncAggregate: &altair.SyncAggregate{
						SyncCommitteeBits:      bitfield.NewBitvector512(),
						SyncCommitteeSignature: phase0.BLSSignature{},
					},
				},
			},
		}}

	events := &Events{
		EtherReceived: []*contract.ContractEtherReceived{
			{
				Sender:         [20]byte{56, 140, 129, 140, 168, 185, 37, 27, 57, 49, 49, 192, 138, 115, 106, 103, 204, 177, 146, 151},
				DonationAmount: big.NewInt(1000),
				Raw: types.Log{
					BlockNumber: 8745218,
					TxHash:      common.Hash{1, 2, 3},
					Topics:      []common.Hash{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
				},
			},
		},
	}

	header := &types.Header{
		Number:     big.NewInt(8745218),
		Difficulty: big.NewInt(8745218),
	}
	receipts := []*types.Receipt{
		{
			TxHash:      common.Hash{1, 2, 3},
			BlockNumber: big.NewInt(8745218),
			Logs: []*types.Log{
				{
					BlockNumber: 8745218,
					TxHash:      common.Hash{1, 2, 3},
					Topics:      []common.Hash{{1, 2, 3}, {1, 2, 3}, {1, 2, 3}},
				},
			},
		},
	}

	// Creates the full block with above data
	fullBlock := NewFullBlock(proposalDuty, validator)
	fullBlock.SetConsensusBlock(block)
	fullBlock.SetEvents(events)
	fullBlock.SetHeaderAndReceipts(header, receipts)

	// Serialize the fullblock
	jsonData, err := json.MarshalIndent(fullBlock, "", " ")
	require.NoError(t, err)

	// This is human readable output
	//log.Info(fmt.Sprintf("Saving oracle state: %s", jsonData))

	// Recover the full block
	var recoveredFullBlock FullBlock
	err = json.Unmarshal(jsonData, &recoveredFullBlock)
	require.NoError(t, err)

	// Assert the recovered fields match the expected
	require.Equal(t, phase0.Slot(5214140), recoveredFullBlock.ConsensusDuty.Slot)
	require.Equal(t, phase0.ValidatorIndex(12), recoveredFullBlock.ConsensusDuty.ValidatorIndex)
	require.Equal(t, big.NewInt(8745218), recoveredFullBlock.ExecutionHeader.Number)
	require.Equal(t, big.NewInt(8745218), recoveredFullBlock.ExecutionReceipts[0].BlockNumber)
	require.Equal(t, big.NewInt(1000), recoveredFullBlock.Events.EtherReceived[0].DonationAmount)
	require.Equal(t, uint64(8745218), recoveredFullBlock.Events.EtherReceived[0].Raw.BlockNumber)
	require.Equal(t, "0x388C818CA8B9251b393131C08a736A67ccB19297", recoveredFullBlock.Events.EtherReceived[0].Sender.String())
	require.Equal(t, "0x388C818CA8B9251b393131C08a736A67ccB19297", recoveredFullBlock.GetFeeRecipient())
	require.Equal(t, spec.DataVersionBellatrix, recoveredFullBlock.ConsensusBlock.Version)

}

func LoadFullBlock(slotNumber uint64, chainId string, hasHeaders bool) (*FullBlock, error) {
	fileName := fmt.Sprintf("fullblock_slot_%d_chainid_%s%s.json", slotNumber, chainId, HasHeader(hasHeaders))
	path := filepath.Join("../mock", fileName)
	jsonFile, err := os.Open(path)
	defer jsonFile.Close()
	if err != nil {
		return nil, errors.Wrap(err, "could not open json file")
	}

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read json file")
	}

	var fullBlock FullBlock

	err = json.Unmarshal(byteValue, &fullBlock)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal json file")
	}

	return &fullBlock, nil
}

func LoadValidators() (map[phase0.ValidatorIndex]*v1.Validator, error) {
	path := filepath.Join("../mock", "validators.json")
	jsonFile, err := os.Open(path)
	defer jsonFile.Close()
	if err != nil {
		return nil, errors.Wrap(err, "could not open json file")
	}

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read json file")
	}

	var validators map[phase0.ValidatorIndex]*v1.Validator

	err = json.Unmarshal(byteValue, &validators)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal json file")
	}

	return validators, nil
}
