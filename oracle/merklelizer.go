package oracle

import (
	"encoding/hex"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"

	solsha3 "github.com/miguelmota/go-solidity-sha3"
	mt "github.com/txaty/go-merkletree"
	"golang.org/x/crypto/sha3"
)

type testData struct {
	data []byte
}

func (t *testData) Serialize() ([]byte, error) {
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
				// In lowercase to avoid confusion when claiming
				DepositAddress: strings.ToLower(validator.DepositAddress),
				// Copy the value
				AccumulatedBalance: new(big.Int).Set(validator.AccumulatedRewardsWei),
			})
		}
	}

	// Run a sanity check to make sure the after the transformations we are distributing
	// the same amount of rewards as the total accumulated rewards
	allAccumulatedFromValidators := big.NewInt(0)
	for _, validator := range state.Validators {
		allAccumulatedFromValidators.Add(allAccumulatedFromValidators, validator.AccumulatedRewardsWei)
	}

	allAccumulatedFromDeposits := big.NewInt(0)
	for _, depositAddressAccumulated := range allLeafs {
		allAccumulatedFromDeposits.Add(allAccumulatedFromDeposits, depositAddressAccumulated.AccumulatedBalance)
	}

	if allAccumulatedFromValidators.Cmp(allAccumulatedFromDeposits) != 0 {
		log.Fatal("rewards calculation per validator and per deposit address does not match: ",
			allAccumulatedFromValidators, " vs ", allAccumulatedFromDeposits)
	}

	// Order the leafs by deposit address
	orderedByDepositAddress := merklelizer.OrderByDepositAddress(allLeafs)

	// Sanity check to ensure the PoolAddress is not already in the link of DepositAddress
	// This should never happen and would be a weird missconfiguration
	for _, leaf := range orderedByDepositAddress {
		if strings.ToLower(leaf.DepositAddress) == strings.ToLower(state.PoolAddress) {
			log.Fatal("the PoolAddress is equal to one of the DepositAddress. ",
				"PoolAddress: ", state.PoolAddress, " DepositAddress: ", leaf.DepositAddress)
		}

	}

	// Prepend the leaf with the pool fees to the list of leafs. Always the first
	poolFeesLeaf := RawLeaf{
		DepositAddress:     strings.ToLower(state.PoolAddress),
		AccumulatedBalance: new(big.Int).Set(state.PoolAccumulatedFees),
	}

	// Pool rewards leaf (address + balance) is the first. Note that the DepositAddress name is reused
	// which could be missleading
	orderedByDepositAddress = append([]RawLeaf{poolFeesLeaf}, orderedByDepositAddress...)

	// Before returning the leaf, ensure all of them are valid addresses
	for _, leaf := range orderedByDepositAddress {
		if !common.IsHexAddress(leaf.DepositAddress) {
			log.Fatal("leaf contained a wrong deposit address: ", leaf.DepositAddress)
		}

		// To avoid compatibility problems, all DepositAddress should be in lowercase
		if strings.ToLower(leaf.DepositAddress) != leaf.DepositAddress {
			log.Fatal("all deposit address should be in lowercase: ", leaf.DepositAddress)
		}
	}

	return orderedByDepositAddress
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
func (merklelizer *Merklelizer) GenerateTreeFromState(state *OracleState) (map[string]mt.DataBlock, map[string]RawLeaf, *mt.MerkleTree, bool) {

	blocks := make([]mt.DataBlock, 0)

	orderedRawLeafs := merklelizer.AggregateValidatorsIndexes(state)

	log.WithFields(log.Fields{
		"Leafs": len(orderedRawLeafs),
	}).Info("Generating tree")

	// TODO: refactor this.
	// Stores the deposit address -> hashed leaf
	depositToLeaf := make(map[string]mt.DataBlock, 0)
	// Stores te deposit address -> raw leaf
	depositToRawLeaf := make(map[string]RawLeaf, 0)

	for _, leaf := range orderedRawLeafs {
		leafHash := solsha3.SoliditySHA3(
			solsha3.Address(leaf.DepositAddress),
			solsha3.Uint256(leaf.AccumulatedBalance),
		)
		blocks = append(blocks, &testData{data: leafHash})
		depositToLeaf[leaf.DepositAddress] = &testData{data: leafHash}
		depositToRawLeaf[leaf.DepositAddress] = leaf

		log.WithFields(log.Fields{
			"DepositAddress":     leaf.DepositAddress,
			"AccumulatedBalance": leaf.AccumulatedBalance,
			"LeafHash":           hex.EncodeToString(leafHash),
		}).Info("Leaf information")

	}

	if len(blocks) < 2 {
		// Returns false meaning that we dont have enough data to generate a merkle tree
		// Expected behaviour at the begining with none or just 1 validator registered
		return nil, nil, nil, false
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

	// TODO: Improve logs, use debug. TODO: unused?
	for i := 0; i < len(blocks); i++ {
		serialized, err := blocks[i].Serialize()
		if err != nil {
			log.Fatal(err)
		}
		_ = serialized
		//log.Info("Proof of block index :", i, " blockhash:  ", hex.EncodeToString(serialized))
		proof0, err := tree.GenerateProof(blocks[i])
		if err != nil {
			log.Fatal(err)
		}
		for j, proof := range proof0.Siblings {
			_ = j
			_ = proof
			//log.Info("proof: ", hex.EncodeToString(proof))
		}
	}

	return depositToLeaf, depositToRawLeaf, tree, true
}
