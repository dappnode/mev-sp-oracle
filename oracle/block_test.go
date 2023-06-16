package oracle

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
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

// TODO: Important test donations
//5862054, //donation normal
//5862104, //donation via smart contract

func Test_GetProposerTip(t *testing.T) {

	type test struct {
		// Input
		Name                 string
		BlockNumber          uint64
		ExpectedTip          *big.Int
		ExpectedFeeRecipient string
	}
	/*

		tests := []test{
			// subscribed validator proposes mev block with correct fee https://prater.beaconcha.in/slot/5739624
			{}
		}

		for _, tt := range tests {
			t.Run(tt.Name, func(t *testing.T) {

				//require.Equal(t, tt.ExpeectedWithCred, block.WithdrawalAddress)
			})
		}*/

}

// Proposer tip of vanila block has to be calculated by adding all manually tips
// there is no field available, and it has to be manually recreated using all tx
// receipts present in that block.
func Test_GetProperTip_Mainnet_Slot_5320341(t *testing.T) {
	// Decode a a hardcoded block/header/receipts
	fileName := "bellatrix_slot_5320341_mainnet"
	block, header, receipts := LoadBlockHeaderReceiptsBellatrix(t, fileName)
	extendedBlock := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionBellatrix, Bellatrix: &block}
	myBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5320341,
		ValidatorIndex: phase0.ValidatorIndex(87961),
	}, &v1.Validator{
		Index: 87961,
	})
	myBlock.SetConsensusBlock(extendedBlock)
	myBlock.SetHeaderAndReceipts(&header, receipts)

	// Get proposer tip
	proposerTip, err := myBlock.GetProposerTip()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1944763730864393), proposerTip)
}

// This block contains a tx with a smart contract deployment
// Tests if the fee is calculated properly as these tx
// are a bit different.
func Test_GetProperTip_Mainnet_Slot_5344344(t *testing.T) {
	fileName := "bellatrix_slot_5344344_mainnet"
	block, header, receipts := LoadBlockHeaderReceiptsBellatrix(t, fileName)
	extendedBlock := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionBellatrix, Bellatrix: &block}
	myBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5344344,
		ValidatorIndex: phase0.ValidatorIndex(356208),
	}, &v1.Validator{
		Index: 356208,
	})
	myBlock.SetConsensusBlock(extendedBlock)
	myBlock.SetHeaderAndReceipts(&header, receipts)

	mevReward, mev, mevFeeRecipient := myBlock.MevRewardInWei()
	require.Equal(t, big.NewInt(99952842017043014), mevReward)
	require.Equal(t, mev, true)
	require.Equal(t, mevFeeRecipient, "0x388c818ca8b9251b393131c08a736a67ccb19297")

	proposerTip, err := myBlock.GetProposerTip()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(95434044627649514), proposerTip)
}

func Test_GetProperTip_Goerli_Slot_5214302(t *testing.T) {
	fileName := "capella_slot_5214302_goerli"
	block, header, receipts := LoadBlockHeaderReceiptsCapella(t, fileName)
	extendedBlock := spec.VersionedSignedBeaconBlock{Version: spec.DataVersionCapella, Capella: &block}
	myBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5214302,
		ValidatorIndex: phase0.ValidatorIndex(218475),
	}, &v1.Validator{
		Index: 218475,
	})
	myBlock.SetConsensusBlock(&extendedBlock)
	myBlock.SetHeaderAndReceipts(&header, receipts)

	proposerTip, err := myBlock.GetProposerTip()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(38657065851824731), proposerTip)
}

func Test_GetMevReward_Goerli_Slot_5214321(t *testing.T) {
	fileName := "capella_slot_5214321_goerli"
	block, header, receipts := LoadBlockHeaderReceiptsCapella(t, fileName)
	extendedBlock := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionCapella, Capella: &block}
	myBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5214321,
		ValidatorIndex: phase0.ValidatorIndex(252922),
	}, &v1.Validator{
		Index: 252922,
	})
	myBlock.SetConsensusBlock(extendedBlock)
	myBlock.SetHeaderAndReceipts(&header, receipts)

	// Gets the MEV reward that was sent to a specific address
	mevReward, mev, mevFeeRecipient := myBlock.MevRewardInWei()
	require.Equal(t, big.NewInt(15867629069461526), mevReward)
	require.Equal(t, mev, true)
	require.Equal(t, mevFeeRecipient, "0x4d496ccc28058b1d74b7a19541663e21154f9c84")

	// This block was a MEV block, but we can also test the tip
	proposerTip, err := myBlock.GetProposerTip()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(15992505660349526), proposerTip)
}

func Test_GetMevReward_Goerli_Slot_5307527(t *testing.T) {
	// This block contains a tx to 0x553bd5a94bcc09ffab6550274d5db140a95ae9bc
	// but its a normal tx not an MEV one. Detect it doesnt produce a false positive
	// https://prater.beaconcha.in/slot/5307527
	fileName := "capella_slot_5307527_goerli"
	block, header, receipts := LoadBlockHeaderReceiptsCapella(t, fileName)
	extendedBlock := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionCapella, Capella: &block}
	myBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5307527,
		ValidatorIndex: phase0.ValidatorIndex(289213),
	}, &v1.Validator{
		Index: 289213,
	})
	myBlock.SetConsensusBlock(extendedBlock)
	myBlock.SetHeaderAndReceipts(&header, receipts)

	// No mev reward
	_, mev, mevFeeRecipient := myBlock.MevRewardInWei()
	require.Equal(t, mev, false)
	require.Equal(t, mevFeeRecipient, "")

	// This block was a MEV block, but we can also test the tip
	proposerTip, err := myBlock.GetProposerTip()
	require.NoError(t, err)
	require.Equal(t, big.NewInt(105735750887810922), proposerTip)
}

func Test_MevReward_Slot_5320342(t *testing.T) {
	fileName := "bellatrix_slot_5320342_mainnet"
	block, _, _ := LoadBlockHeaderReceiptsBellatrix(t, fileName)
	extendedBlock := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionBellatrix, Bellatrix: &block}
	myBlock := NewFullBlock(&v1.ProposerDuty{
		Slot:           5320342,
		ValidatorIndex: phase0.ValidatorIndex(42156),
	}, &v1.Validator{
		Index: 42156,
	})
	myBlock.SetConsensusBlock(extendedBlock)

	// Check that mev reward is correct and sent to the address
	mevReward, mev, mevFeeRecipient := myBlock.MevRewardInWei()
	require.Equal(t, big.NewInt(65184406499820485), mevReward)
	require.Equal(t, mev, true)
	require.Equal(t, mevFeeRecipient, "0xf8636377b7a998b51a3cf2bd711b870b3ab0ad56")
}

func Test_Marashal(t *testing.T) {

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

// Util to load from file
func LoadBlockHeaderReceiptsBellatrix(t *testing.T, file string) (bellatrix.SignedBeaconBlock, types.Header, []*types.Receipt) {
	blockJson, err := os.Open("../mock/block_" + file)
	require.NoError(t, err)
	blockByte, err := ioutil.ReadAll(blockJson)
	require.NoError(t, err)
	var bellatrixblock bellatrix.SignedBeaconBlock
	err = bellatrixblock.UnmarshalJSON(blockByte)
	require.NoError(t, err)

	var headerBlock types.Header
	headerJson, err := os.Open("../mock/header_" + file)
	headerByte, err := ioutil.ReadAll(headerJson)
	err = headerBlock.UnmarshalJSON(headerByte)
	require.NoError(t, err)

	var txReceipts []*types.Receipt
	txReceiptsJson, err := os.Open("../mock/txreceipts_" + file)
	txReceiptsByte, err := ioutil.ReadAll(txReceiptsJson)
	err = json.Unmarshal(txReceiptsByte, &txReceipts)
	require.NoError(t, err)

	return bellatrixblock, headerBlock, txReceipts
}

func LoadBlockHeaderReceiptsCapella(t *testing.T, file string) (capella.SignedBeaconBlock, types.Header, []*types.Receipt) {
	blockJson, err := os.Open("../mock/block_" + file)
	require.NoError(t, err)
	blockByte, err := ioutil.ReadAll(blockJson)
	require.NoError(t, err)
	var capellaBlock capella.SignedBeaconBlock
	err = capellaBlock.UnmarshalJSON(blockByte)
	require.NoError(t, err)

	var headerBlock types.Header
	headerJson, err := os.Open("../mock/header_" + file)
	require.NoError(t, err)
	fmt.Println("jeader", headerJson)
	headerByte, err := ioutil.ReadAll(headerJson)
	require.NoError(t, err)
	err = headerBlock.UnmarshalJSON(headerByte)
	require.NoError(t, err)

	var txReceipts []*types.Receipt
	txReceiptsJson, err := os.Open("../mock/txreceipts_" + file)
	txReceiptsByte, err := ioutil.ReadAll(txReceiptsJson)
	err = json.Unmarshal(txReceiptsByte, &txReceipts)
	require.NoError(t, err)

	return capellaBlock, headerBlock, txReceipts
}
