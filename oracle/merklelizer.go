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
	DepositAddress     string
	AccumulatedBalance *big.Int
}

// TODO: Add checks:
// -New balance to claim matches what was sent to the pool, etc.

// Aggregates all validators indexes that belong to the same deposit address. This
// allows the merkle tree to hold all validators balance belonging to the same set
// of validators, that makes claiming cheaper since only one proof is needed for n validators
// belonging to the same deposit address
func (merklelizer *Merklelizer) AggregateValidatorsIndexes(state *OracleState) []RawLeaf {

	// Creates an array of leaf. Each leaf contains the deposit address and the accumulated balance
	// for all the validators belonging to the same deposit address
	allLeafs := make([]RawLeaf, 0)

	// Iterate all validators
	for _, validator := range state.Validators {

		// That match some criteria
		if validator.ValidatorStatus != Banned && validator.ValidatorStatus != NotSubscribed {
			found := false

			// If the leaf already exists, add the balance to the existing leaf (by deposit address)
			for _, leaf := range allLeafs {
				if leaf.DepositAddress == validator.DepositAddress {
					leaf.AccumulatedBalance.Add(leaf.AccumulatedBalance, validator.AccumulatedRewardsWei)
					found = true
					continue
				}
			}

			// If the leaf does not exist, create a new one, initing the balance to the current validator balance
			if !found {
				allLeafs = append(allLeafs, RawLeaf{
					DepositAddress:     validator.DepositAddress,
					AccumulatedBalance: new(big.Int).Set(validator.AccumulatedRewardsWei), // Copy the value
				})
			}
		}
	}

	// Run a sanity check to make sure the after the transformations we are distributing
	// the same amount of rewards as the total accumulated rewards
	allAccumulatedFromValidators := big.NewInt(0)
	for _, validator := range state.Validators {
		if validator.ValidatorStatus != Banned && validator.ValidatorStatus != NotSubscribed {
			allAccumulatedFromValidators.Add(allAccumulatedFromValidators, validator.AccumulatedRewardsWei)
		}
	}

	allAccumulatedFromDeposits := big.NewInt(0)
	for _, depositAddressAccumulated := range allLeafs {
		allAccumulatedFromDeposits.Add(allAccumulatedFromDeposits, depositAddressAccumulated.AccumulatedBalance)
	}

	if allAccumulatedFromValidators.Cmp(allAccumulatedFromDeposits) != 0 {
		log.Fatal("rewards calculation per validator and per deposit address does not match: ",
			allAccumulatedFromValidators, " vs ", allAccumulatedFromDeposits)
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
func (merklelizer *Merklelizer) GenerateTreeFromState(state *OracleState) (map[string]mt.DataBlock, map[string]RawLeaf, *mt.MerkleTree) {

	blocks := make([]mt.DataBlock, 0)

	orderedRawLeafs := merklelizer.AggregateValidatorsIndexes(state)

	log.Info("orderedRawLeafs", orderedRawLeafs)

	// TODO: refactor this.
	// Stores the deposit address -> hashed leaf
	depositToLeaf := make(map[string]mt.DataBlock, 0)
	// Stores te deposit address -> raw leaf
	depositToRawLeaf := make(map[string]RawLeaf, 0)

	for _, leaf := range orderedRawLeafs {
		// TODO: Improve logs and move to debug
		log.Info("leaf.DepositAddress: ", leaf.DepositAddress)
		log.Info("leaf.ClaimableBalance: ", leaf.AccumulatedBalance)

		leafHash := solsha3.SoliditySHA3(
			solsha3.Address(leaf.DepositAddress),
			solsha3.Uint256(leaf.AccumulatedBalance),
		)
		log.Info("leafHash: ", hex.EncodeToString(leafHash), " Deposit addres: ", leaf.DepositAddress)
		blocks = append(blocks, &testData{data: leafHash})
		depositToLeaf[leaf.DepositAddress] = &testData{data: leafHash}
		depositToRawLeaf[leaf.DepositAddress] = leaf
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

	return depositToLeaf, depositToRawLeaf, tree

	// TODO: Update contract with root.
	// TODO: Generate dump with proofs.
}
