package oracle

import "github.com/attestantio/go-eth2-client/spec/capella"

// TODO: Once Capella spec is finalized.
// Most likely it won't change much and we can inherit block_bellatrix.go

type CapellaBlock struct {
	capella.SignedBeaconBlock
}
