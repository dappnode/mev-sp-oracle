package oracle

import (
	"encoding/hex"
	"math/big"
	"sort"

	log "github.com/sirupsen/logrus"

	solsha3 "github.com/miguelmota/go-solidity-sha3"
	mt "github.com/txaty/go-merkletree"
	"golang.org/x/crypto/sha3"
)

type testData struct {
	data []byte
}

func (t *testData) Serialize() ([]byte, error) {
	log.Info("serializing", hex.EncodeToString(t.data))
	return t.data, nil
}

func KeccakHash(data []byte) ([]byte, error) {
	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	return hash.Sum(nil), nil
}

type Merklelizer struct {
}

func NewMerklelizer() *Merklelizer {
	merklelizer := &Merklelizer{}
	return merklelizer
}

type RawLeaf struct {
	DepositAddress   string
	PoolRecipient    string
	ClaimableBalance *big.Int
	UnbanBalance     *big.Int
}

// TODO: Add checks:
// -New balance to claim matches what was sent to the pool, etc.

// Aggregates all validators indexes that belong to the same pool recipient
// AND deposit address.
// TODO: This requires further testing
func (merklelizer *Merklelizer) AggregateValidatorsIndexes(state *OracleState) []RawLeaf {
	// Start all validators to be unprocessed
	processed := make(map[uint64]bool)
	for valIndex, _ := range state.PoolRecipientAddresses {
		processed[valIndex] = false
	}

	allLeafs := make([]RawLeaf, 0)

	// TODO: toLowerCase
	// TODO Check sizes len(state.PendingRewards), len(state.ClaimableRewards), len(state.UnbanBalances), len(state.DepositAddresses), len(state.PoolRecipientAddresses)

	// Iterate all validators
	for valIndex, _ := range state.PoolRecipientAddresses {
		poolRecipient := state.PoolRecipientAddresses[valIndex]
		depositAddress := state.DepositAddresses[valIndex]

		poolRecipientClaimableBalance := new(big.Int).SetUint64(0)
		poolRecipientUnbanBalance := new(big.Int).SetUint64(0)

		// Validator already processed
		if processed[valIndex] {
			continue
		}

		// Iterate all validators again and check if they have the same pool recipient and deposit address
		for valIndex2, _ := range state.PoolRecipientAddresses {
			poolRecipient2 := state.PoolRecipientAddresses[valIndex2]
			depositAddress2 := state.DepositAddresses[valIndex2]

			if poolRecipient == poolRecipient2 && depositAddress == depositAddress2 {
				// flag as processed
				processed[valIndex2] = true

				poolRecipientClaimableBalance.Add(poolRecipientClaimableBalance, state.ClaimableRewards[valIndex2])
				poolRecipientUnbanBalance.Add(poolRecipientUnbanBalance, state.UnbanBalances[valIndex2])
			}
		}
		allLeafs = append(allLeafs, RawLeaf{
			DepositAddress:   depositAddress,
			PoolRecipient:    poolRecipient,
			ClaimableBalance: poolRecipientClaimableBalance,
			UnbanBalance:     poolRecipientUnbanBalance,
		})
	}

	// Add one by one the ones that could not be aggregated (unprocessed)
	for unprocIndex, _ := range processed {
		if processed[unprocIndex] {
			continue
		}
		allLeafs = append(allLeafs, RawLeaf{
			DepositAddress:   state.DepositAddresses[unprocIndex],
			PoolRecipient:    state.PoolRecipientAddresses[unprocIndex],
			ClaimableBalance: state.ClaimableRewards[unprocIndex],
			UnbanBalance:     state.UnbanBalances[unprocIndex],
		})
	}
	return merklelizer.OrderByDepositAddress(allLeafs)
}

// Sort by deposit address
func (merklelizer *Merklelizer) OrderByDepositAddress(leafs []RawLeaf) []RawLeaf {
	sortedLeafs := make([]RawLeaf, len(leafs))
	copy(sortedLeafs, leafs)
	sort.Slice(sortedLeafs, func(i, j int) bool {
		return sortedLeafs[i].DepositAddress < sortedLeafs[j].DepositAddress
	})
	return sortedLeafs
}

// return map of deposit address -> and its hashed leaf. rethink this
func (merklelizer *Merklelizer) GenerateTreeFromState(state *OracleState) (map[string]mt.DataBlock, *mt.MerkleTree) {

	blocks := make([]mt.DataBlock, 0)

	orderedRawLeafs := merklelizer.AggregateValidatorsIndexes(state)

	log.Info("orderedRawLeafs", orderedRawLeafs)

	depositToLeaf := make(map[string]mt.DataBlock, 0)

	for _, leaf := range orderedRawLeafs {
		// TODO: Improve logs and move to debug
		log.Info("leaf.DepositAddress: ", leaf.DepositAddress)
		log.Info("leaf.PoolRecipient: ", leaf.PoolRecipient)
		log.Info("leaf.ClaimableBalance: ", leaf.ClaimableBalance)
		log.Info("leaf.UnbanBalance: ", leaf.UnbanBalance)
		leafHash := solsha3.SoliditySHA3(
			solsha3.Address(leaf.DepositAddress),
			solsha3.Address(leaf.PoolRecipient),
			solsha3.Uint256(leaf.ClaimableBalance),
			solsha3.Uint256(leaf.UnbanBalance),
		)
		log.Info("leafHash: ", hex.EncodeToString(leafHash), " Deposit addres: ", leaf.DepositAddress)
		blocks = append(blocks, &testData{data: leafHash})
		depositToLeaf[leaf.DepositAddress] = &testData{data: leafHash}
	}

	if len(blocks) <= 1 {
		// TODO handle this.
		log.Fatal("TODO: cant generate tree with less than 2 blocks")
	}

	tree, err := mt.New(&mt.Config{
		SortSiblingPairs: true,
		HashFunc:         KeccakHash,
		Mode:             mt.ModeTreeBuild,
		DoNotHashLeaves:  true,
	}, blocks)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Improve logs, use debug
	for i := 0; i < len(blocks); i++ {
		serrr, err := blocks[i].Serialize()
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Proof of block index :", i, " blockhash:  ", hex.EncodeToString(serrr))
		proof0, err := tree.GenerateProof(blocks[i])
		if err != nil {
			log.Fatal(err)
		}
		for j, proof := range proof0.Siblings {
			_ = j
			log.Info("proof: ", hex.EncodeToString(proof))
		}
	}

	return depositToLeaf, tree

	// TODO: Update contract with root.
	// TODO: Generate dump with proofs.
}
