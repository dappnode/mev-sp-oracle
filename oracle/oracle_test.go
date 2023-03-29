package oracle

import (
	"testing"

	"github.com/dappnode/mev-sp-oracle/config"
	"github.com/stretchr/testify/require"
)

// Todo: perhaps move. e2e test, requiere a beacon node.
func Test_EndToEnd_VanilaReward(t *testing.T) {
	t.Skip("Skipping e2e test")
	var cfg = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfg)
	require.NoError(t, err)
	oracle := NewOracle(&config.Config{
		PoolAddress:           "0xffee087852cb4898e6c3532e776e68bc68b1143b",
		CheckPointSizeInSlots: 5,
		DeployedSlot:          5344344,
	}, onchain)
	oracle.State.LatestSlot = oracle.cfg.DeployedSlot - 1
	slot, err := oracle.AdvanceStateToNextSlot()
	require.NoError(t, err)
	require.Equal(t, oracle.cfg.DeployedSlot, slot)
	//log.Info("checkpoint", checkpointInfo)

	// TODO: all these test are unfinished!
	_ = oracle
}

func Test_EndToEnd_MevReward(t *testing.T) {
	t.Skip("Skipping e2e test")
	var cfg = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfg)
	require.NoError(t, err)
	oracle := NewOracle(&config.Config{
		PoolAddress:           "0x4675c7e5baafbffbca748158becba61ef3b0a263",
		CheckPointSizeInSlots: 5,
		DeployedSlot:          5323502,
	}, onchain)
	//checkpointInfo, err := oracle.CalculateCheckpointRewards(0)
	//require.NoError(t, err)
	//log.Info("checkpoint", checkpointInfo)
	_ = oracle
}

// TODO: test slot: 5323601
// it contained a contract deployment that was crashing the code msg.To() apprear to be nil
func Test_EndToEnd_NoSubscriptions(t *testing.T) {
	t.Skip("Skipping e2e test")
	var cfg = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	onchain, err := NewOnchain(cfg)
	require.NoError(t, err)
	oracle := NewOracle(&config.Config{
		PoolAddress:           "0x4675c7e5baafbffbca748158becba61ef3b0a263",
		CheckPointSizeInSlots: 100,
		DeployedSlot:          5323601,
	}, onchain)
	oracle.State.LatestSlot = 5323600
	slot, err := oracle.AdvanceStateToNextSlot()
	require.Equal(t, oracle.cfg.DeployedSlot, slot)
	require.NoError(t, err)
	//log.Info("checkpoint", checkpointInfo)
	_ = oracle
}

func Test_TODO(t *testing.T) {
	/*
		oracle := Oracle{
			smoothingPoolAddress:  "address",
			checkpointSizeInSlots: 1,
			deployedBlock:         1,
		}
		signedBeaconBlock := &spec.VersionedSignedBeaconBlock{
			Version: spec.DataVersionBellatrix,
			Bellatrix: &bellatrix.SignedBeaconBlock{
				Message: &bellatrix.BeaconBlock{
					Slot:          5214140,
					ProposerIndex: 0,
					ParentRoot:    [32]byte{}, //remove?
					StateRoot:     [32]byte{}, // remove?
					Body: &bellatrix.BeaconBlockBody{
						ExecutionPayload: &bellatrix.ExecutionPayload{
							FeeRecipient: [20]byte{},
							BlockNumber:  0,
							Transactions: []bellatrix.Transaction{
								{1, 2},
								{1, 2}},
						},
					},
				},
				//Signature: [], not needed?
			},
		}
	*/
	//log.Info("----", oracle.IsRewardOurs(signedBeaconBlock))
}
