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
	var onchain = NewOnchain(cfg)
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
	var onchain = NewOnchain(cfg)
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
	var onchain = NewOnchain(cfg)
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

func Test_IsValidatorSubscribed(t *testing.T) {
	t.Skip("Skipping e2e test")
	var cfg = config.Config{
		ConsensusEndpoint: "http://127.0.0.1:5051",
		ExecutionEndpoint: "http://127.0.0.1:8545",
	}
	var onchain = NewOnchain(cfg)
	_ = onchain
	//oracle := NewOracle(&config.Config{}, onchain)
	//var subscriptions = Subscriptions{
	// TODO: missing many fields in here
	//subscriptions: map[uint64]string{
	//		481020: "0x", // propose mev block at 5323504
	//			168929: "0x", // proposes vanila block at 5323506
	//		},
	//	}
	//is481020 := oracle.IsValidatorSubscribed(481020, &subscriptions)
	//is168929 := oracle.IsValidatorSubscribed(168929, &subscriptions)
	//is100000 := oracle.IsValidatorSubscribed(100000, &subscriptions)
	//is400000 := oracle.IsValidatorSubscribed(400000, &subscriptions)

	//require.Equal(t, is481020, true)
	//require.Equal(t, is168929, true)
	//require.Equal(t, is100000, false)
	//require.Equal(t, is400000, false)
}
