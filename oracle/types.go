package oracle

// active subscriptions, fetched from the smart contract
type Subscriptions struct {
	blockHeigh string
	slotHeigh  string
	// TODO: unsure abou this: validator->controlledAddress?
	subscriptions map[uint64]string //start, end, etc. see smart contract.
}
